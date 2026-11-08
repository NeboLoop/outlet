package workers

import (
	"context"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/svc"
)

// OutboundWorker runs the outbound engine: a send pass every 30 seconds,
// reply reading every 2 minutes, and the DNS recheck every day.
type OutboundWorker struct {
	svcCtx *svc.ServiceContext
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// StartOutboundWorker starts the engine. Outbound does nothing for brands
// that aren't in the server's Outbound.Brands config.
func StartOutboundWorker(svcCtx *svc.ServiceContext) *OutboundWorker {
	ctx, cancel := context.WithCancel(context.Background())
	w := &OutboundWorker{svcCtx: svcCtx, cancel: cancel}
	w.every(ctx, 30*time.Second, svcCtx.Outbound.Tick)
	w.every(ctx, 2*time.Minute, svcCtx.Outbound.PollReplies)
	w.every(ctx, 24*time.Hour, svcCtx.Outbound.CheckAllDomains)
	return w
}

func (w *OutboundWorker) every(ctx context.Context, d time.Duration, fn func(context.Context)) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		t := time.NewTicker(d)
		defer t.Stop()
		for {
			fn(ctx)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

// Stop stops the engine and waits for the current pass to finish.
func (w *OutboundWorker) Stop() {
	w.cancel()
	w.wg.Wait()
}
