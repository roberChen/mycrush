package app

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/pubsub"
)

// TestSetupSubscriber_PublishBeforeGoroutineScheduled deterministically
// reproduces the subscriber-before-publisher race that exists when
// setupSubscriber calls Subscribe *inside* the goroutine.
//
// With GOMAXPROCS=1 the goroutine launched by setupSubscriber cannot
// run until the calling goroutine blocks. So if we Publish to the
// source broker immediately after setupSubscriber returns:
//
//   - Buggy code (Subscribe in goroutine): src has zero subscribers
//     at Publish time → event silently dropped → timeout.
//   - Fixed code (Subscribe before goroutine): src already has the
//     fan-in channel registered → event is buffered → forwarded when
//     the goroutine eventually runs.
func TestSetupSubscriber_PublishBeforeGoroutineScheduled(t *testing.T) {
	// Pin to a single OS thread so goroutine scheduling is
	// deterministic: launched goroutines cannot preempt the caller
	// until it blocks.
	prev := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(prev)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	src := pubsub.NewBroker[string]()
	defer src.Shutdown()
	out := pubsub.NewBroker[tea.Msg]()
	defer out.Shutdown()

	// Subscribe to the output broker synchronously so it is
	// registered before we publish.
	outCh := out.Subscribe(ctx)

	var wg sync.WaitGroup
	setupSubscriber(ctx, &wg, "test", src.Subscribe, out)

	// Publish immediately — no yield, no sleep. With GOMAXPROCS=1
	// the fan-in goroutine has NOT been scheduled yet.
	src.Publish(pubsub.CreatedEvent, "race-event")

	// Block: the scheduler now runs the fan-in goroutine. With the
	// fix it finds the buffered event and forwards it to out.
	select {
	case <-outCh:
		// Event was forwarded — the fan-in goroutine had already
		// subscribed to src before we published.
	case <-time.After(2 * time.Second):
		t.Fatal("event lost: setupSubscriber's goroutine had not subscribed when Publish was called (race confirmed)")
	}

	cancel()
	wg.Wait()
}
