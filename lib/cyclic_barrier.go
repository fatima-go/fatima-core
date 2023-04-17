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
	"sync"
	"sync/atomic"
)

type CyclicBarrier struct {
	generation int
	count      int32
	parties    int32
	trigger    *sync.Cond
	wgControl  *sync.WaitGroup
	trip       func()
}

func NewCyclicBarrier(parties int, trip func()) *CyclicBarrier {
	wgControl := &sync.WaitGroup{}
	defer wgControl.Wait()

	b := CyclicBarrier{}
	b.wgControl = &sync.WaitGroup{}
	b.count = int32(parties)
	b.parties = int32(parties)
	b.trigger = sync.NewCond(&sync.Mutex{})
	b.trip = trip
	return &b
}

func (this *CyclicBarrier) waitUntil() int {
	this.trigger.L.Lock()
	generation := this.generation
	c := atomic.AddInt32(&this.count, int32(-1))

	if c == 0 {
		if this.trip != nil {
			this.trip()
		}
		this.trigger.Broadcast()
		this.count = this.parties
		this.generation++
		this.trigger.L.Unlock()
		return this.generation
	}
	//...
	for generation == this.generation {
		this.trigger.Wait()
	}
	this.trigger.L.Unlock()
	return this.generation
}

func (this *CyclicBarrier) getCount() int {
	return int(atomic.LoadInt32(&this.count))
}

func (this *CyclicBarrier) checkCount(c int) bool {
	if int(atomic.LoadInt32(&this.count)) == c {
		return true
	}
	return false
}

func (this *CyclicBarrier) Wait() {
	this.wgControl.Wait()
}

func (this *CyclicBarrier) Dispatch(f func()) {
	this.wgControl.Add(1)
	go func() {
		this.process(f)
		this.wgControl.Done()
	}()
}

func (this *CyclicBarrier) process(f func()) {
	f()
	this.waitUntil()
}
