package kworx

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/multierr"
)

type DoWithValueFunc func(string) error

func NewRunner(threadiness int, values []string, fn DoWithValueFunc) *runner {
	return &runner{
		threadiness: threadiness,
		fn:          fn,
		values: values,
	}
}

type AtomicBool struct {
	flag int32
}

func (b *AtomicBool) Set() {
	var i int32 = 1
	atomic.StoreInt32(&(b.flag), int32(i))
}

func (b *AtomicBool) Get() bool {
	if atomic.LoadInt32(&(b.flag)) != 0 {
		return true
	}
	return false
}

type runner struct {
	interrupted AtomicBool
	fn          DoWithValueFunc
	threadiness int
	values []string
}

func (r *runner) Run(stopCh <-chan struct{}) error {
	// unbuffered so that we block when all workers are busy
	valueCh := make(chan string)
	errCh := make(chan error)

	var errs []error
	// this must be a pointer
	errWG := &sync.WaitGroup{}
	errWG.Add(1)
	go func() {
		defer errWG.Done()
		for err := range errCh {
			if err != nil {
				errs = append(errs, err)
			}
		}
	}()

	go func() {
		<-stopCh
		r.interrupted.Set()
	}()

	// this must be a pointer
	wg := &sync.WaitGroup{}
	for w := 0; w < r.threadiness; w++ {
		wg.Add(1)
		go r.worker(r.fn, valueCh, errCh, wg)
	}

	for _, value := range r.values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		valueCh <- value
	}
	close(valueCh)
	wg.Wait()
	close(errCh)
	errWG.Wait()
	if r.interrupted.Get() {
		errs = append(errs, fmt.Errorf("interrupted"))
	}
	return multierr.Combine(errs...)
}

func (r *runner) worker(fn DoWithValueFunc, valueCh <-chan string, errCh chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	for value := range valueCh {
		if r.interrupted.Get() {
			return
		}
		err := fn(value)
		if err != nil {
			// wrap it with restConfig for extra context
			err = newRunnerError(value, err)
		}
		errCh <- err
	}
}

func newRunnerError(value string, cause error) error {
	return fmt.Errorf("[value %q]: %w", value, cause)
}
