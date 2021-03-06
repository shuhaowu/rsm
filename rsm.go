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

// Creates a handler that transitions to another state. Typically called after
// the transition has taken place (`StageAfter`).
//
// Takes a string that's the next state's name and returns an EventHandler.
//
// Example:
//   sm.AddTransition([]string{"start"}, "middle", nil)
//   sm.AddTransition([]string{"middle"}, "end", nil)
//   sm.AddAfterHandler([]string{"start"}, "middle", rsm.TransitionTo("end"))
//
//   sm.Transit("middle") // will go all the way to the end.
func TransitionTo(state string) EventHandler {
	return func(e *Event) error {
		return e.RSM.Transit(state, e.Args...)
	}
}

type transitionKey struct {
	startState string
	endState   string
	stage      int
}

type RSM struct {
	transitions        map[transitionKey][]EventHandler
	beforeTransition   EventHandler
	finalizeTransition EventHandler
	afterTransition    EventHandler
	quit               chan bool

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
	rsm.transitions = make(map[transitionKey][]EventHandler)

	return rsm
}

func (r *RSM) BeforeTransitionHandler(handler EventHandler) {
	r.beforeTransition = handler
}

func (r *RSM) FinalizeTransitionHandler(handler EventHandler) {
	r.finalizeTransition = handler
}

func (r *RSM) AfterTransitionHandler(handler EventHandler) {
	r.afterTransition = handler
}

func (r *RSM) AddHandler(startStates []string, endState string, stage int, handler EventHandler) {
	for _, startState := range startStates {
		handlers, ok := r.transitions[transitionKey{startState, endState, stage}]
		if ok {
			handlers = append(handlers, handler)
		} else {
			handlers = []EventHandler{handler}
		}
		r.transitions[transitionKey{startState, endState, stage}] = handlers
	}
}

func (r *RSM) AddBeforeHandler(startStates []string, endState string, handler EventHandler) {
	r.AddHandler(startStates, endState, StageBefore, handler)
}

func (r *RSM) AddAfterHandler(startStates []string, endState string, handler EventHandler) {
	r.AddHandler(startStates, endState, StageAfter, handler)
}

func (r *RSM) AddInProgressHandler(startStates []string, endState string, handler EventHandler) {
	if handler == nil {
		handler = NilHandler
	}

	r.AddHandler(startStates, endState, StageInProgress, handler)
}

func (r *RSM) AddTransition(startStates []string, endState string, handler EventHandler) {
	r.AddInProgressHandler(startStates, endState, handler)
}

func (r *RSM) CanTransitionTo(state string) bool {
	_, ok := r.transitions[transitionKey{r.CurrentState, state, StageInProgress}]
	return ok
}

func (r *RSM) Transit(nextState string, args ...interface{}) error {
	if !r.CanTransitionTo(nextState) {
		return errors.New(fmt.Sprintf("Cannot transition from %s to %s", r.CurrentState, nextState))
	}

	var handlers []EventHandler
	var ok bool
	var event *Event
	var err error

	if r.beforeTransition != nil {
		event = &Event{
			RSM:   r,
			Stage: StageBefore,
			Src:   r.CurrentState,
			Dest:  nextState,
			Args:  args,
		}
		err = r.beforeTransition(event)
		if err != nil {
			return err
		}
	}

	// Before transition handler
	handlers, ok = r.transitions[transitionKey{r.CurrentState, nextState, StageBefore}]

	if ok {
		event = &Event{
			RSM:   r,
			Stage: StageBefore,
			Src:   r.CurrentState,
			Dest:  nextState,
			Args:  args,
		}
		for _, handler := range handlers {
			err = handler(event)
			if err != nil {
				return err
			}
		}
	}

	// Event transition handler
	handlers, _ = r.transitions[transitionKey{r.CurrentState, nextState, StageInProgress}]
	event = &Event{
		RSM:   r,
		Stage: StageInProgress,
		Src:   r.CurrentState,
		Dest:  nextState,
		Args:  args,
	}
	for _, handler := range handlers {
		err = handler(event)
		if err != nil {
			return err
		}
	}

	if r.finalizeTransition != nil {
		event = &Event{
			RSM:   r,
			Stage: StageInProgress,
			Src:   r.CurrentState,
			Dest:  nextState,
			Args:  args,
		}
		err := r.finalizeTransition(event)
		if err != nil {
			return err
		}
	}

	beforeState := r.CurrentState
	r.CurrentState = nextState

	// After transition handler
	handlers, ok = r.transitions[transitionKey{beforeState, r.CurrentState, StageAfter}]
	if ok {
		event = &Event{
			RSM:   r,
			Stage: StageAfter,
			Src:   beforeState,
			Dest:  r.CurrentState,
			Args:  args,
		}
		// After transition handler must not return an error.
		for _, handler := range handlers {
			handler(event)
		}
	}

	if r.afterTransition != nil {
		event = &Event{
			RSM:   r,
			Stage: StageAfter,
			Src:   beforeState,
			Dest:  r.CurrentState,
			Args:  args,
		}
		r.afterTransition(event)
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
