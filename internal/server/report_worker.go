package server

import "context"

type parallelBackgroundWorker struct{ first, second contextWorker }

func (w *parallelBackgroundWorker) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); w.second.Run(ctx) }()
	w.first.Run(ctx)
	cancel()
	<-done
}

type scheduledReportWorker struct{ store *reportScheduleStore }

func (w *scheduledReportWorker) Run(ctx context.Context) {
	w.store.Run(ctx, func() error { return nil })
}
