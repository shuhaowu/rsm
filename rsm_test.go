package rsm

import (
	"errors"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func retryWaitTime(i int) time.Duration {
	return time.Millisecond
}

func Test(t *testing.T) { TestingT(t) }

type RSMSuite struct {
	rsm *RSM
	err error
}

var _ = Suite(&RSMSuite{})

func (r *RSMSuite) SetUpTest(c *C) {
	r.rsm = NewRSM("start", retryWaitTime, 100)
	r.err = errors.New("failed")
}

func (r *RSMSuite) TestStateTranstionBefore(c *C) {
	args := []string{"1", "2"}
	handlerCalled := false

	handler := func(e *Event) error {
		handlerCalled = true
		eargs := make([]string, len(e.Args))
		for i, a := range e.Args {
			eargs[i] = a.(string)
		}

		c.Assert(eargs, DeepEquals, args)
		c.Assert(e.Src, Equals, "start")
		c.Assert(e.Dest, Equals, "end")
		c.Assert(e.RSM, Equals, r.rsm)
		c.Assert(e.Stage, Equals, StageBefore)
		c.Assert(e.RSM.CurrentState, Equals, "start")
		return nil
	}

	r.rsm.AddTransition("start", "end", nil)
	r.rsm.AddHandler("start", "end", StageBefore, handler)

	err := r.rsm.Transit("end", "1", "2")
	c.Assert(err, IsNil)
	c.Assert(handlerCalled, Equals, true)
	c.Assert(r.rsm.CurrentState, Equals, "end")
}

func (r *RSMSuite) TestStateTransitionInProgress(c *C) {
	args := []string{"1", "2"}
	handlerCalled := false

	handler := func(e *Event) error {
		handlerCalled = true
		eargs := make([]string, len(e.Args))
		for i, a := range e.Args {
			eargs[i] = a.(string)
		}

		c.Assert(eargs, DeepEquals, args)
		c.Assert(e.Src, Equals, "start")
		c.Assert(e.Dest, Equals, "end")
		c.Assert(e.RSM, Equals, r.rsm)
		c.Assert(e.Stage, Equals, StageInProgress)
		c.Assert(e.RSM.CurrentState, Equals, "start")
		return nil
	}

	r.rsm.AddTransition("start", "end", handler)

	err := r.rsm.Transit("end", "1", "2")
	c.Assert(err, IsNil)
	c.Assert(handlerCalled, Equals, true)
	c.Assert(r.rsm.CurrentState, Equals, "end")
}

func (r *RSMSuite) TestStateTransitionAfter(c *C) {
	args := []string{"1", "2"}
	handlerCalled := false

	handler := func(e *Event) error {
		handlerCalled = true
		eargs := make([]string, len(e.Args))
		for i, a := range e.Args {
			eargs[i] = a.(string)
		}

		c.Assert(eargs, DeepEquals, args)
		c.Assert(e.Src, Equals, "start")
		c.Assert(e.Dest, Equals, "end")
		c.Assert(e.RSM, Equals, r.rsm)
		c.Assert(e.Stage, Equals, StageAfter)
		c.Assert(e.RSM.CurrentState, Equals, "end")
		return nil
	}

	r.rsm.AddTransition("start", "end", nil)
	r.rsm.AddHandler("start", "end", StageAfter, handler)

	err := r.rsm.Transit("end", "1", "2")
	c.Assert(err, IsNil)
	c.Assert(handlerCalled, Equals, true)
	c.Assert(r.rsm.CurrentState, Equals, "end")
}

func (r *RSMSuite) TestStateTransitionFailDuringBefore(c *C) {
	handler := func(e *Event) error {
		return r.err
	}

	r.rsm.AddTransition("start", "end", nil)
	r.rsm.AddHandler("start", "end", StageBefore, handler)

	err := r.rsm.Transit("end")
	c.Assert(err, Equals, r.err)
	c.Assert(r.rsm.CurrentState, Equals, "start")
}

func (r *RSMSuite) TestStateTransitionFailDuringInProgress(c *C) {
	handler := func(e *Event) error {
		return r.err
	}

	r.rsm.AddTransition("start", "end", handler)

	err := r.rsm.Transit("end")
	c.Assert(err, Equals, r.err)
	c.Assert(r.rsm.CurrentState, Equals, "start")
}

func (r *RSMSuite) TestBeforeAfterFinalizeAllTransitionsHandler(c *C) {
	args := []string{"1", "2"}
	beforeHandlerCalled := false
	finalizeHandlerCalled := false
	afterHandlerCalled := false
	beforeHandler := func(e *Event) error {
		beforeHandlerCalled = true
		eargs := make([]string, len(e.Args))
		for i, a := range e.Args {
			eargs[i] = a.(string)
		}

		c.Assert(eargs, DeepEquals, args)
		c.Assert(e.Src, Equals, "start")
		c.Assert(e.Dest, Equals, "end")
		c.Assert(e.RSM, Equals, r.rsm)
		c.Assert(e.Stage, Equals, StageBefore)
		c.Assert(e.RSM.CurrentState, Equals, "start")
		return nil
	}

	finalizeHandler := func(e *Event) error {
		finalizeHandlerCalled = true
		eargs := make([]string, len(e.Args))
		for i, a := range e.Args {
			eargs[i] = a.(string)
		}

		c.Assert(eargs, DeepEquals, args)
		c.Assert(e.Src, Equals, "start")
		c.Assert(e.Dest, Equals, "end")
		c.Assert(e.RSM, Equals, r.rsm)
		c.Assert(e.Stage, Equals, StageInProgress)
		c.Assert(e.RSM.CurrentState, Equals, "start")
		return nil
	}

	afterHandler := func(e *Event) error {
		afterHandlerCalled = true
		eargs := make([]string, len(e.Args))
		for i, a := range e.Args {
			eargs[i] = a.(string)
		}

		c.Assert(eargs, DeepEquals, args)
		c.Assert(e.Src, Equals, "start")
		c.Assert(e.Dest, Equals, "end")
		c.Assert(e.RSM, Equals, r.rsm)
		c.Assert(e.Stage, Equals, StageAfter)
		c.Assert(e.RSM.CurrentState, Equals, "end")
		return nil
	}

	r.rsm.BeforeTransitionHandler(beforeHandler)
	r.rsm.FinalizeTransitionHandler(finalizeHandler)
	r.rsm.AfterTransitionHandler(afterHandler)
	r.rsm.AddTransition("start", "end", nil)

	err := r.rsm.Transit("end", "1", "2")
	c.Assert(err, IsNil)
	c.Assert(beforeHandlerCalled, Equals, true)
	c.Assert(finalizeHandlerCalled, Equals, true)
	c.Assert(afterHandlerCalled, Equals, true)
	c.Assert(r.rsm.CurrentState, Equals, "end")
}

func (r *RSMSuite) TestMultipleHandlers(c *C) {
	counter := 0
	handler := func(e *Event) error {
		counter++
		return nil
	}

	r.rsm.AddTransition("start", "end", handler)
	r.rsm.AddTransition("start", "end", handler)
	r.rsm.AddTransition("start", "end", handler)
	r.rsm.AddTransition("start", "end", handler)
	r.rsm.AddTransition("start", "end", handler)
	r.rsm.AddTransition("start", "end", handler)
	err := r.rsm.Transit("end")
	c.Assert(err, IsNil)
	c.Assert(counter, Equals, 6)
}

func (r *RSMSuite) TestMultipleHandlersFailAny(c *C) {
	successCalled := 0
	handler := func(e *Event) error {
		successCalled++
		return nil
	}

	failHandler := func(e *Event) error {
		return r.err
	}

	r.rsm.AddTransition("start", "end", handler)
	r.rsm.AddTransition("start", "end", handler)
	r.rsm.AddTransition("start", "end", failHandler)
	r.rsm.AddTransition("start", "end", handler)

	err := r.rsm.Transit("end")
	c.Assert(err, Equals, r.err)
	c.Assert(successCalled, Equals, 2)
}

func (r *RSMSuite) TestTransitionToNonExistentState(c *C) {
	err := r.rsm.Transit("wat")
	c.Assert(err, NotNil)
}

func (r *RSMSuite) TestHandlerOrders(c *C) {
	stage := 0
	beforeAllHandler := func(e *Event) error {
		c.Assert(stage, Equals, 0)
		stage = 1
		return nil
	}

	beforeTransitionHandler := func(e *Event) error {
		c.Assert(stage, Equals, 1)
		stage = 2
		return nil
	}

	inProgressHandler1 := func(e *Event) error {
		c.Assert(stage, Equals, 2)
		stage = 3
		return nil
	}

	inProgressHandler2 := func(e *Event) error {
		c.Assert(stage, Equals, 3)
		stage = 4
		return nil
	}

	afterTransitionHandler := func(e *Event) error {
		c.Assert(stage, Equals, 4)
		stage = 5
		return nil
	}

	afterAllHandler := func(e *Event) error {
		c.Assert(stage, Equals, 5)
		stage = 6
		return nil
	}

	r.rsm.BeforeTransitionHandler(beforeAllHandler)
	r.rsm.AddHandler("start", "end", StageBefore, beforeTransitionHandler)
	r.rsm.AddTransition("start", "end", inProgressHandler1)
	r.rsm.AddTransition("start", "end", inProgressHandler2)
	r.rsm.AddHandler("start", "end", StageAfter, afterTransitionHandler)
	r.rsm.AfterTransitionHandler(afterAllHandler)

	r.rsm.Transit("end")
	c.Assert(stage, Equals, 6)
}

func (r *RSMSuite) TestStateTransitionRetries(c *C) {
	failHandler := func(e *Event) error {
		return r.err
	}

	i := 0
	successAfter3 := func(e *Event) error {
		i++
		if i < 3 {
			return r.err
		}
		return nil
	}

	r.rsm.AddTransition("start", "fail", failHandler)
	r.rsm.AddTransition("start", "end", nil)
	r.rsm.AddHandler("start", "end", StageBefore, successAfter3)

	err := r.rsm.TransitWithRetries("fail")
	c.Assert(err, NotNil)
	c.Assert(r.rsm.CurrentState, Equals, "start")

	err = r.rsm.TransitWithRetries("end")
	c.Assert(err, IsNil)
	c.Assert(i, Equals, 3)
	c.Assert(r.rsm.CurrentState, Equals, "end")
}
