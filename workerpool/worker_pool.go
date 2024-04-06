package workerpool

import (
	"context"
	"sync"
)

type Task struct {
	Func func(args ...interface{}) *Result
	Args []interface{}
}

type Result struct {
	Value interface{}
	Err   error
}

type WorkerPool interface {
	Start(ctx context.Context)
	Tasks() chan *Task
	Results() chan *Result
}

type workerPool struct {
	numWorkers int
	tasks      chan *Task
	results    chan *Result
	wg         *sync.WaitGroup
}

var _ WorkerPool = (*workerPool)(nil)

func NewWorkerPool(numWorkers int, bufferSize int) *workerPool {
	return &workerPool{
		numWorkers: numWorkers,
		tasks:      make(chan *Task, bufferSize),
		results:    make(chan *Result, bufferSize),
		wg:         &sync.WaitGroup{},
	}
}

func (wp *workerPool) Start(ctx context.Context) {
    for i := 0; i < wp.numWorkers; i++ {
        wp.wg.Add(1)
        go func() {
            wp.run(ctx)
            wp.wg.Done()
        }()
    }

    // Wait for all workers to finish
    wp.wg.Wait()
    // Close the results channel as no more results will be sent
    close(wp.results)
}

func (wp *workerPool) Tasks() chan *Task {
	return wp.tasks
}

func (wp *workerPool) Results() chan *Result {
	return wp.results
}

func (wp *workerPool) run(ctx context.Context) {
    for {
        select {
        case task, ok := <-wp.tasks:
            if !ok {
                // Task channel is closed, return from this worker
                return
            }
            // Execute the task and send the result to the results channel
            result := task.Func(task.Args...)
            select {
            case wp.results <- result:
                // Result sent successfully
            case <-ctx.Done():
                // Context was cancelled, return without sending the result
                return
            }
        case <-ctx.Done():
            // Context was cancelled, return from this worker
            return
        }
    }
}