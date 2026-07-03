package notification

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Service — the public interface consumed by business services
// ─────────────────────────────────────────────────────────────────────────────

// Service is the only interface business services (leaveflow, employee, auth)
// depend on. They call Publish() and never know about workers, channels, or email.
//
// This interface is also trivially mockable in tests.
type Service interface {
	// Publish enqueues a notification event for asynchronous processing.
	// It never blocks the caller — if the channel is full the event is dropped
	// and a warning is logged (back-pressure protection).
	Publish(event Event)

	// Start launches the worker pool. Call once at application startup.
	Start(ctx context.Context)

	// Stop drains in-flight events and shuts down workers gracefully.
	// Blocks until all workers have exited or the deadline is reached.
	Stop()
}

// ─────────────────────────────────────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────────────────────────────────────

// Config controls the worker pool behaviour.
type Config struct {
	// Workers is the number of concurrent event-processing goroutines.
	// Default: 3. Tune based on email API rate limits.
	Workers int

	// BufferSize is the capacity of the event channel.
	// Default: 256. Events beyond this are dropped with a warning.
	BufferSize int

	// MaxRetries is the number of retry attempts for a failed delivery.
	// Default: 3.
	MaxRetries int

	// RetryBaseDelay is the initial back-off before the first retry.
	// Subsequent retries double the delay (exponential back-off).
	// Default: 500ms.
	RetryBaseDelay time.Duration
}

func DefaultConfig() Config {
	return Config{
		Workers:        3,
		BufferSize:     256,
		MaxRetries:     3,
		RetryBaseDelay: 500 * time.Millisecond,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// dispatcher — concrete implementation
// ─────────────────────────────────────────────────────────────────────────────

type dispatcher struct {
	processor *EventProcessor
	events    chan Event
	cfg       Config
	logger    *slog.Logger
	wg        sync.WaitGroup
	once      sync.Once // guards Stop()
	stopCh    chan struct{}
}

// NewService constructs a dispatcher that satisfies the Service interface.
//
// Usage in main.go:
//
//	svc := notification.NewService(processor, notification.DefaultConfig(), logger)
//	svc.Start(ctx)
//	defer svc.Stop()
func NewService(processor *EventProcessor, cfg Config, logger *slog.Logger) Service {
	if cfg.Workers <= 0 {
		cfg.Workers = 3
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 256
	}
	return &dispatcher{
		processor: processor,
		events:    make(chan Event, cfg.BufferSize),
		cfg:       cfg,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// Publish enqueues an event without blocking the caller.
// If the buffer is full the event is dropped and a warning is logged.
// Dropping is intentional: notification delivery is best-effort and must
// never stall a business transaction.
func (d *dispatcher) Publish(event Event) {
	select {
	case d.events <- event:
		d.logger.Debug("notification: event enqueued", "type", event.Type)
	default:
		d.logger.Warn("notification: channel full, event dropped",
			"type", event.Type,
			"buffer_size", d.cfg.BufferSize,
		)
	}
}

// Start launches cfg.Workers goroutines. Safe to call only once.
func (d *dispatcher) Start(ctx context.Context) {
	d.logger.Info("notification: starting worker pool", "workers", d.cfg.Workers)

	for i := range d.cfg.Workers {
		d.wg.Add(1)
		go d.worker(ctx, i+1)
	}
}

// Stop signals workers to drain and exit, then waits with a 30-second deadline.
func (d *dispatcher) Stop() {
	d.once.Do(func() {
		d.logger.Info("notification: shutting down worker pool")
		close(d.stopCh) // signal workers to stop accepting new events
		close(d.events) // let workers drain the remaining buffered events

		done := make(chan struct{})
		go func() {
			d.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			d.logger.Info("notification: all workers exited cleanly")
		case <-time.After(30 * time.Second):
			d.logger.Warn("notification: shutdown timeout — some events may be lost")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// worker — one goroutine per worker index
// ─────────────────────────────────────────────────────────────────────────────

// worker reads events from the channel until it is closed, then exits.
// It handles each event with retry logic so transient email failures are
// recovered without losing the event.
func (d *dispatcher) worker(ctx context.Context, id int) {
	defer d.wg.Done()
	d.logger.Debug("notification: worker started", "id", id)

	for event := range d.events {
		// Respect context cancellation between events
		select {
		case <-ctx.Done():
			d.logger.Warn("notification: worker context cancelled, stopping", "id", id)
			return
		default:
		}
		d.processWithRetry(ctx, event, id)
	}

	d.logger.Debug("notification: worker exited", "id", id)
}

// processWithRetry calls Process() up to cfg.MaxRetries times with exponential
// back-off. Errors inside handlers are caught via recover() so one bad event
// never kills a worker goroutine.
func (d *dispatcher) processWithRetry(ctx context.Context, event Event, workerID int) {
	delay := d.cfg.RetryBaseDelay

	for attempt := 1; attempt <= d.cfg.MaxRetries; attempt++ {
		err := d.safeProcess(event)
		if err == nil {
			return
		}

		if attempt == d.cfg.MaxRetries {
			d.logger.Error("notification: max retries exceeded, event dropped",
				"type", event.Type,
				"worker", workerID,
				"attempts", attempt,
				"err", err,
			)
			return
		}

		d.logger.Warn("notification: delivery failed, retrying",
			"type", event.Type,
			"worker", workerID,
			"attempt", attempt,
			"retry_in", delay,
			"err", err,
		)

		// Exponential back-off — respect context cancellation during sleep
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			delay *= 2
		}
	}
}

// safeProcess wraps Process() with recover() so a handler panic never kills
// the worker goroutine. It converts panics into errors for the retry logic.
func (d *dispatcher) safeProcess(event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("notification: handler panicked",
				"type", event.Type,
				"panic", r,
			)
			err = errFromPanic(r)
		}
	}()
	d.processor.Process(event)
	return nil
}

// errFromPanic converts a recover() value into an error.
func errFromPanic(r any) error {
	if e, ok := r.(error); ok {
		return e
	}
	return fmt.Errorf("panic: %v", r)
}
