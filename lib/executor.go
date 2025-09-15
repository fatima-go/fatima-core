/*
 * Copyright 2023 github.com/fatima-go
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @project fatima-core
 * @author dave
 * @date 23. 4. 12. 오후 1:41
 */

package lib

import (
	"context"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/fatima-go/fatima-log"
)

type Executor interface {
	Insert(val interface{})
	Cancel()
	Wait()
	Close()
	Count() int
}

func NewExecutorBuilder(executeFunc func(interface{})) ExecutorBuilder {
	wb := new(executorBuilder)
	wb.executeFunc = executeFunc
	wb.workerSize = runtime.NumCPU()
	return wb
}

type ExecutorBuilder interface {
	SetQueueSize(int) ExecutorBuilder
	SetWorkerSize(int) ExecutorBuilder
	Build() Executor
}

type executorBuilder struct {
	queueSize   int
	workerSize  int
	wg          *sync.WaitGroup
	executeFunc func(interface{})
}

func (wb *executorBuilder) SetQueueSize(size int) ExecutorBuilder {
	if size > 0 {
		wb.queueSize = size
	}
	return wb
}

func (wb *executorBuilder) SetWorkerSize(size int) ExecutorBuilder {
	if size > 0 {
		wb.workerSize = size
	}
	return wb
}

func (wb *executorBuilder) Build() Executor {
	if wb.executeFunc == nil {
		log.Error("executeFunc is nil")
		return nil
	}

	ctx := new(executor)
	ctx.innerCtx, ctx.cancel = context.WithCancel(context.Background())
	ctx.queue = make(chan interface{}, wb.queueSize)

	for w := 0; w < wb.workerSize; w++ {
		go ctx.startExecute(w, wb.executeFunc)
	}

	return ctx
}

type executor struct {
	innerCtx context.Context
	cancel   context.CancelFunc
	queue    chan interface{}
	wg       sync.WaitGroup
	count    uint32
}

func (w *executor) startExecute(workerId int, executeFunc func(interface{})) {
	log.Trace("executor worker %d started", workerId)
	for true {
		select {
		case event := <-w.queue:
			if event == nil {
				continue
			}
			log.Trace("[%d] executor worker event", workerId)
			w.fetch(executeFunc, event)
		case <-w.innerCtx.Done():
			log.Trace("[%d] executor worker finished", workerId)
			return
		}
	}
}

func (w *executor) fetch(executeFunc func(interface{}), val interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic to execute", r)
			log.Error("%s", string(debug.Stack()))
		}
	}()
	defer func() {
		atomic.AddUint32(&w.count, 1)
		w.wg.Done()
	}()
	executeFunc(val)
}

func (w *executor) Insert(val interface{}) {
	w.wg.Add(1)
	w.queue <- val
}

func (w *executor) Cancel() {
	w.cancel()
}

func (w *executor) Wait() {
	w.wg.Wait()
	w.cancel()
}

func (w *executor) Close() {
	close(w.queue)
}

func (w *executor) Count() int {
	return int(w.count)
}
