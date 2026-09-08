package operations

import (
	"context"
	"sync"
	"time"
)

// Lifecycle coordinates long-running backend work with app shutdown.
type Lifecycle struct {
	rootCtx    context.Context
	cancelRoot context.CancelFunc
	timeout    time.Duration

	mu           sync.Mutex
	shuttingDown bool
	wg           sync.WaitGroup
}

func NewLifecycle(timeout time.Duration) *Lifecycle {
	rootCtx, cancelRoot := context.WithCancel(context.Background())
	return &Lifecycle{
		rootCtx:    rootCtx,
		cancelRoot: cancelRoot,
		timeout:    timeout,
	}
}

func (l *Lifecycle) OperationContext() (context.Context, func()) {
	if l == nil {
		return context.Background(), func() {}
	}

	l.mu.Lock()
	track := !l.shuttingDown
	if track {
		l.wg.Add(1)
	}
	l.mu.Unlock()

	ctx := l.rootCtx
	var cancel context.CancelFunc = func() {}
	if l.timeout > 0 {
		ctx, cancel = context.WithTimeout(l.rootCtx, l.timeout)
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			cancel()
			if track {
				l.wg.Done()
			}
		})
	}
	return ctx, release
}

func (l *Lifecycle) Shutdown(ctx context.Context) error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	if !l.shuttingDown {
		l.shuttingDown = true
		l.cancelRoot()
	}
	l.mu.Unlock()

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
