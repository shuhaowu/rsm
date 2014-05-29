package rsm

import (
	"errors"
	"fmt"
	"time"
)

const (
	StageBefore = iota
	StageInProgress
	StageAfter
)

type Event struct {
	RSM   *RSM
	Stage int
	Src   string
	Dest  string
	Args  []interface{}
}

type EventHandler func(*Event) error

func NilHandler(e *Event) error {
	return nil
}

type transitionKey struct {
	startState string
	endState   string
	stage      int
}

type RSM struct {
	transitions map[transitionKey]EventHandler
	quit        chan bool

	CurrentState  string
	RetryWaitTime func(int) time.Duration
	MaxRetries    int
	Parent        interface{}
}

func NewRSM(currentState string, retriesWaitTime func(int) time.Duration, maxRetries int) *RSM {
	rsm := &RSM{}
	rsm.CurrentState = currentState
	rsm.RetryWaitTime = retriesWaitTime
	rsm.MaxRetries = maxRetries
	rsm.quit = make(chan bool)
	rsm.transitions = make(map[transitionKey]EventHandler)

	return rsm
}

func (r *RSM) AddHandler(startState, endState string, stage int, handler EventHandler) {
	r.transitions[transitionKey{startState, endState, stage}] = handler
}

func (r *RSM) AddTransition(startState, endState string, handler EventHandler) {
	if handler == nil {
		handler = NilHandler
	}

	r.AddHandler(startState, endState, StageInProgress, handler)
}

func (r *RSM) CanTransitionTo(state string) bool {
	_, ok := r.transitions[transitionKey{r.CurrentState, state, StageInProgress}]
	return ok
}

func (r *RSM) Transit(nextState string, args ...interface{}) error {
	if !r.CanTransitionTo(nextState) {
		return errors.New(fmt.Sprintf("Cannot transition from %s to %s", r.CurrentState, nextState))
	}

	var handler EventHandler
	var ok bool
	var event *Event
	var err error

	// Before transition handler
	handler, ok = r.transitions[transitionKey{r.CurrentState, nextState, StageBefore}]

	if ok {
		event = &Event{
			RSM:   r,
			Stage: StageBefore,
			Src:   r.CurrentState,
			Dest:  nextState,
			Args:  args,
		}
		err = handler(event)
		if err != nil {
			return err
		}
	}

	// Event transition handler
	handler, _ = r.transitions[transitionKey{r.CurrentState, nextState, StageInProgress}]
	event = &Event{
		RSM:   r,
		Stage: StageInProgress,
		Src:   r.CurrentState,
		Dest:  nextState,
		Args:  args,
	}
	err = handler(event)
	if err != nil {
		return err
	}

	beforeState := r.CurrentState
	r.CurrentState = nextState

	// After transition handler
	handler, ok = r.transitions[transitionKey{beforeState, r.CurrentState, StageAfter}]
	if ok {
		event = &Event{
			RSM:   r,
			Stage: StageAfter,
			Src:   beforeState,
			Dest:  r.CurrentState,
			Args:  args,
		}
		// After transition handler must not return an error.
		handler(event)
	}
	return nil
}

func (r *RSM) maxRetriesReached(nextState string, err error) error {
	return errors.New(fmt.Sprintf("Error transitioning from %s to %s with error: %v", r.CurrentState, nextState, err))
}

func (r *RSM) TransitWithRetries(nextState string, args ...interface{}) error {
	var err error
	i := 1

	for {
		select {
		case <-r.quit:
			return err
		case <-time.After(r.RetryWaitTime(i)):
			if i > r.MaxRetries {
				return r.maxRetriesReached(nextState, err)
			}

			err = r.Transit(nextState, args...)
			if err == nil {
				return nil
			}

			i++
		}
	}
}

func (r *RSM) Stop() {
	r.quit <- true
}
