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
	Go(func() error)
	Wait() error
}

type group struct {
	ctxCancel func()

	wg sync.WaitGroup
	c  chan error
}

func New() Group {
	return &group{c: make(chan error)}
}

// WithContext returns a new Group and an associated Context derived from ctx.
//
// The derived Context is cancelled the first time a function passed to Go
// returns a nil error or when all of them return, whichever occurs first.
func WithContext(ctx context.Context) (Group, context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	return &group{ctxCancel: cancel, c: make(chan error)}, ctx
}

// Wait blocks until the first function calls from the Go method has returned
// with a `nil` error.
func (g *group) Wait() error {
	go func() {
		g.wg.Wait()
		close(g.c)
	}()

	var lastErr error
	for err := range g.c {
		if err != nil {
			lastErr = err
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

		return nil
	}

	// At this point, all `Go` functions have returned

	if g.ctxCancel != nil {
		g.ctxCancel()
	}

	return lastErr
}

// Go calls the given function in a new goroutine.
func (g *group) Go(f func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		g.c <- f()
	}()
}
