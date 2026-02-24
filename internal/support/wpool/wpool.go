// Package wpool provides a lightweight fixed-size worker pool.
package wpool

import (
	"sync"
	"sync/atomic"
)

// Task is the task interface.
type Task interface {
	Run()
}

type funcTask struct {
	fn func()
}

func (f *funcTask) Run() { f.fn() }

// Func wraps a plain func as a Task.
func Func(fn func()) Task { return &funcTask{fn: fn} }

// Pool is a fixed-size goroutine pool with sharded channels.
type Pool struct {
	shards  []chan Task
	done    chan struct{}
	size    uint64
	cursor  atomic.Uint64
	wg      sync.WaitGroup
	closed  atomic.Bool
	OnPanic func(any)
}

// New creates a worker pool.
func New(size, queueSize int) *Pool {
	if size <= 0 {
		size = 1
	}
	if queueSize <= 0 {
		queueSize = 256
	}
	p := &Pool{
		shards: make([]chan Task, size),
		done:   make(chan struct{}),
		size:   uint64(size),
	}
	p.wg.Add(size)
	for i := 0; i < size; i++ {
		p.shards[i] = make(chan Task, queueSize)
		go p.worker(p.shards[i])
	}
	return p
}

func (p *Pool) worker(ch <-chan Task) {
	defer p.wg.Done()
	for {
		select {
		case t, ok := <-ch:
			if !ok {
				return
			}
			p.safeRun(t)
		case <-p.done:
			return
		}
	}
}

func (p *Pool) safeRun(t Task) {
	if p.OnPanic != nil {
		defer func() {
			if r := recover(); r != nil {
				p.OnPanic(r)
			}
		}()
	}
	t.Run()
}

// Submit submits a task to the pool.
func (p *Pool) Submit(t Task) bool {
	if p.closed.Load() {
		return false
	}
	idx := p.cursor.Add(1) % p.size
	select {
	case p.shards[idx] <- t:
		return true
	default:
		return false
	}
}

// SubmitFunc submits a plain function as a task.
func (p *Pool) SubmitFunc(fn func()) bool {
	return p.Submit(Func(fn))
}

// Stop shuts down the pool and waits for all workers to finish.
func (p *Pool) Stop() {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	close(p.done)
	for _, ch := range p.shards {
		close(ch)
	}
	p.wg.Wait()
}
