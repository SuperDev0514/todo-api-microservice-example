package service

import (
	"context"
	"time"

	"github.com/mercari/go-circuitbreaker"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/MarioCarrion/todo-api/internal"
)

// TaskRepository defines the datastore handling persisting Task records.
type TaskRepository interface {
	Create(ctx context.Context, description string, priority internal.Priority, dates internal.Dates) (internal.Task, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (internal.Task, error)
	Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error
}

// TaskSearchRepository defines the datastore handling searching Task records.
type TaskSearchRepository interface {
	Search(ctx context.Context, args internal.SearchArgs) (internal.SearchResults, error)
}

// TaskMessageBrokerRepository defines the datastore handling persisting Searchable Task records.
type TaskMessageBrokerRepository interface {
	Created(ctx context.Context, task internal.Task) error
	Deleted(ctx context.Context, id string) error
	Updated(ctx context.Context, task internal.Task) error
}

// Task defines the application service in charge of interacting with Tasks.
type Task struct {
	repo      TaskRepository
	search    TaskSearchRepository
	msgBroker TaskMessageBrokerRepository
	cb        *circuitbreaker.CircuitBreaker
}

// NewTask ...
func NewTask(logger *zap.Logger, repo TaskRepository, search TaskSearchRepository, msgBroker TaskMessageBrokerRepository) *Task {
	return &Task{
		repo:      repo,
		search:    search,
		msgBroker: msgBroker,
		cb: circuitbreaker.New(
			circuitbreaker.WithOpenTimeout(time.Minute*2),
			circuitbreaker.WithTripFunc(circuitbreaker.NewTripFuncConsecutiveFailures(3)),
			circuitbreaker.WithOnStateChangeHookFn(func(oldState, newState circuitbreaker.State) {
				logger.Info("state changed",
					zap.String("old", string(oldState)),
					zap.String("new", string(newState)),
				)
			}),
		),
	}
}

// By searches Tasks matching the received values.
func (t *Task) By(ctx context.Context, args internal.SearchArgs) (_ internal.SearchResults, err error) {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.By")
	defer span.End()

	if !t.cb.Ready() {
		return internal.SearchResults{}, internal.NewErrorf(internal.ErrorCodeUnknown, "service not available")
	}

	defer func() {
		err = t.cb.Done(ctx, err)
	}()

	res, err := t.search.Search(ctx, args)
	if err != nil {
		return internal.SearchResults{}, internal.WrapErrorf(err, internal.ErrorCodeUnknown, "search")
	}

	return res, nil
}

// Create stores a new record.
//nolint: lll
func (t *Task) Create(ctx context.Context, description string, priority internal.Priority, dates internal.Dates) (internal.Task, error) {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Create")
	defer span.End()

	// XXX: We will revisit the number of received arguments in future episodes.
	task, err := t.repo.Create(ctx, description, priority, dates)
	if err != nil {
		return internal.Task{}, internal.WrapErrorf(err, internal.ErrorCodeUnknown, "repo.Create")
	}

	// XXX: Transactions will be revisited in future episodes.
	_ = t.msgBroker.Created(ctx, task) // XXX: Ignoring errors on purpose

	return task, nil
}

// Delete removes an existing Task from the datastore.
func (t *Task) Delete(ctx context.Context, id string) error {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Delete")
	defer span.End()

	// XXX: We will revisit the number of received arguments in future episodes.
	if err := t.repo.Delete(ctx, id); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnknown, "Delete")
	}

	// XXX: Transactions will be revisited in future episodes.
	_ = t.msgBroker.Deleted(ctx, id) // XXX: Ignoring errors on purpose

	return nil
}

// Task gets an existing Task from the datastore.
func (t *Task) Task(ctx context.Context, id string) (internal.Task, error) {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Task")
	defer span.End()

	// XXX: We will revisit the number of received arguments in future episodes.
	task, err := t.repo.Find(ctx, id)
	if err != nil {
		return internal.Task{}, internal.WrapErrorf(err, internal.ErrorCodeUnknown, "Find")
	}

	return task, nil
}

// Update updates an existing Task in the datastore.
//nolint: lll
func (t *Task) Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Update")
	defer span.End()

	// XXX: We will revisit the number of received arguments in future episodes.
	if err := t.repo.Update(ctx, id, description, priority, dates, isDone); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnknown, "repo.Update")
	}

	{
		// XXX: This will be improved when Kafka events are introduced in future episodes
		task, err := t.repo.Find(ctx, id)
		if err == nil {
			// XXX: Transactions will be revisited in future episodes.
			_ = t.msgBroker.Updated(ctx, task) // XXX: Ignoring errors on purpose
		}
	}

	return nil
}
