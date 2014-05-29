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
