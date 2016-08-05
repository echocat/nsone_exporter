package utils

import (
	"runtime"
	"sync"
)

type WorkerPool struct {
	futures chan *WorkerFuture
}

func NewWorkerPool(numberOfWorkers int, queueSize int) *WorkerPool {
	futures := make(chan *WorkerFuture, queueSize)
	result := &WorkerPool{
		futures: futures,
	}
	runtime.SetFinalizer(result, finalizeWorkerPool)
	for i := 0; i < numberOfWorkers; i++ {
		go result.worker()
	}
	return result
}

func (instance *WorkerPool) Close() {
	close(instance.futures)
}

func (instance *WorkerPool) Submit(task WorkerTask) *WorkerFuture {
	future := NewWorkerFutureFor(task)
	instance.futures <- future
	return future
}

func finalizeWorkerPool(instance *WorkerPool) {
	instance.Close()
}

func (instance *WorkerPool) worker() {
	for future := range instance.futures {
		instance.handle(future)
	}
}

func (instance *WorkerPool) handle(future *WorkerFuture) {
	future.Execute()
}

type WorkerTask func() error

type WorkerFuture struct {
	Func      WorkerTask
	condition *sync.Cond
	done      bool
	err       error
}

type WorkerFutures []*WorkerFuture

func (instance *WorkerFuture) Execute() error {
	instance.condition.L.Lock()
	defer func() {
		instance.done = true
		instance.condition.Broadcast()
		instance.condition.L.Unlock()
	}()
	instance.err = instance.Func()
	return instance.err
}

func (instance *WorkerFutures) Submit(pool *WorkerPool, task WorkerTask) *WorkerFuture {
	future := pool.Submit(task)
	instance.Append(future)
	return future
}

func (instance *WorkerFutures) Append(future *WorkerFuture) *WorkerFutures {
	(*instance) = append(*instance, future)
	return instance
}

func (instance *WorkerFutures) Wait() error {
	for _, future := range *instance {
		err := future.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

func NewWorkerFutureFor(task WorkerTask) *WorkerFuture {
	return &WorkerFuture{
		Func: task,
		condition: &sync.Cond{
			L: &sync.Mutex{},
		},
	}
}

func (instance *WorkerFuture) Wait() error {
	instance.condition.L.Lock()
	defer instance.condition.L.Unlock()
	if !instance.done {
		instance.condition.Wait()
	}
	return instance.err
}
