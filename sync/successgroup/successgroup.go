// Copyright 2018 Luís Fonseca.
//
// Use of this source code is governed by a MIT license that can be found in
// the LICENSE file.

// Package successgroup is the counter-part of `golang.org/x/sync/errgroup`.
// Instead of returning when an error occurs, it instead returns
// **immediately** when there's a successful result. This can be used to issue
// the same request to several different sources and just use the first result.
package successgroup

import (
	"sync"

	"context"
)

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
type Group interface {
	Go(func() (interface{}, error))
	Wait() (interface{}, error)
}

type funcResult struct {
	value interface{}
	err   error
}

type group struct {
	ctxCancel func()

	wg sync.WaitGroup
	c  chan funcResult
}

func New() Group {
	return &group{c: make(chan funcResult)}
}

// WithContext returns a new Group and an associated Context derived from ctx.
//
// The derived Context is cancelled the first time a function passed to Go
// returns a nil error or when all of them return, whichever occurs first.
func WithContext(ctx context.Context) (Group, context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	return &group{ctxCancel: cancel, c: make(chan funcResult)}, ctx
}

// Wait blocks until the first function calls from the Go method has returned
// with a `nil` error.
func (g *group) Wait() (interface{}, error) {
	go func() {
		g.wg.Wait()
		close(g.c)
	}()

	var lastErr error
	for res := range g.c {
		if res.err != nil {
			lastErr = res.err
			continue
		}

		if g.ctxCancel != nil {
			g.ctxCancel()
		}

		go func() {
			for range g.c {
				// drain any messages
			}
		}()

		return res.value, nil
	}

	// At this point, all `Go` functions have returned

	if g.ctxCancel != nil {
		g.ctxCancel()
	}

	return nil, lastErr
}

// Go calls the given function in a new goroutine.
func (g *group) Go(f func() (interface{}, error)) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		value, err := f()
		g.c <- funcResult{
			value: value,
			err:   err,
		}
	}()
}
