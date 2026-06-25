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

package infra

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/lib"
	"github.com/fatima-go/fatima-log"
)

var compPreInit []fatima.FatimaComponent
var compGeneral []fatima.FatimaComponent
var compReader []fatima.FatimaComponent
var compWriter []fatima.FatimaComponent

func init() {
	compPreInit = make([]fatima.FatimaComponent, 0)
	compGeneral = make([]fatima.FatimaComponent, 0)
	compReader = make([]fatima.FatimaComponent, 0)
	compWriter = make([]fatima.FatimaComponent, 0)
}

func registerComponent(comp fatima.FatimaComponent) {
	if c, ok := comp.(fatima.FatimaComponentTypeOrder); ok {
		switch c.GetType() {
		case fatima.COMP_PRE_INIT:
			compPreInit = append(compPreInit, comp)
		case fatima.COMP_GENERAL:
			compGeneral = append(compGeneral, comp)
		case fatima.COMP_READER:
			compReader = append(compReader, comp)
		case fatima.COMP_WRITER:
			compWriter = append(compWriter, comp)
		}
	} else {
		compGeneral = append(compGeneral, comp)
	}
}

// initializeComponent : initialize registered FatimaComponent
func initializeComponent() (res bool) {
	res = false

	defer func() {
		if r := recover(); r != nil {
			log.Error("**PANIC** while initializing", errors.New(fmt.Sprintf("%s", r)))
			log.Error("%s", string(debug.Stack()))
			return
		}
		//res = true
	}()

	if !callInitial(compPreInit) {
		return
	}
	if !callInitial(compWriter) {
		return
	}
	if !callInitial(compReader) {
		return
	}
	if !callInitial(compGeneral) {
		return
	}

	res = true
	return
}

func callInitial(list []fatima.FatimaComponent) bool {
	for _, v := range list {
		if !v.Initialize() {
			return false
		}
	}
	return true
}

func bootupNotify() {
	all := make([]fatima.FatimaComponent, 0)
	all = append(all, compPreInit...)
	all = append(all, compGeneral...)
	all = append(all, compReader...)
	all = append(all, compWriter...)

	size := len(all)
	if size > 0 {
		cyBarrier := lib.NewCyclicBarrier(size, func() { log.Info("process start up successfully") })
		for _, v := range all {
			t := v
			cyBarrier.Dispatch(func() { t.Bootup() })
		}
		cyBarrier.Wait()
	} else {
		log.Info("process start up successfully")
	}
}

// componentGroupTimeout bounds how long a single lifecycle group (goaway or
// shutdown) may take before the orchestrator stops waiting and proceeds to the
// next group. This prevents a hung component from blocking the whole sequence.
var componentGroupTimeout = 4 * time.Second

// shutdownComponent shuts down registered FatimaComponent in data-flow order.
//
// Unlike Initialize (PRE_INIT -> WRITER -> READER -> GENERAL), shutdown is NOT a
// plain reverse. Ingress (READER) is stopped first so no new work enters a
// tearing-down pipeline, then workers (GENERAL) drain, then egress (WRITER)
// flushes, and finally base infrastructure (PRE_INIT) is released:
//
//	Shutdown : READER -> GENERAL -> WRITER -> PRE_INIT
//
// Members within a group run concurrently, but groups are sequential. Each group
// is bounded by componentGroupTimeout. Note this ordering only yields a true
// graceful shutdown when READER components honor the contract
// "stop accepting -> wait in-flight -> return" in their Shutdown().
func shutdownComponent(program string) {
	log.Info("start shutdown FatimaComponent")

	defer func() {
		if r := recover(); r != nil {
			log.Error("**PANIC** while shutdown", errors.New(fmt.Sprintf("%s", r)))
			log.Error("%s", string(debug.Stack()))
			log.Close()
			return
		}
	}()

	shutdownGroup("reader", compReader)
	shutdownGroup("general", compGeneral)
	shutdownGroup("writer", compWriter)
	shutdownGroup("preinit", compPreInit)

	log.Warn("shutdown %s", program)
	log.Close()
}

// shutdownGroup shuts down every component in a single type group concurrently and
// waits up to componentGroupTimeout for completion. A panic in one component is
// isolated and does not affect the others.
func shutdownGroup(name string, list []fatima.FatimaComponent) {
	if len(list) == 0 {
		return
	}

	done := make(chan struct{})
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(len(list))
		for _, v := range list {
			t := v
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						log.Error("**PANIC** while shutdown [%s] component : %s", name, r)
						log.Error("%s", string(debug.Stack()))
					}
				}()
				t.Shutdown()
			}()
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("shutdown group [%s] %d component(s) done", name, len(list))
	case <-time.After(componentGroupTimeout):
		log.Warn("shutdown group [%s] timeout after %v, proceeding to next group", name, componentGroupTimeout)
	}
}

// goawayComponent broadcasts Goaway() to components that implement
// FatimaRuntimeGoaway, in the same data-flow group order as shutdown
// (READER -> GENERAL -> WRITER -> PRE_INIT). Members within a group run
// concurrently; groups are sequential and bounded by componentGroupTimeout.
// Type does not restrict participation: any component implementing the interface
// is signaled, it only controls the order.
func goawayComponent() {
	log.Info("start calling goaway...")

	defer func() {
		if r := recover(); r != nil {
			log.Warn("**PANIC** while goaway", errors.New(fmt.Sprintf("%s", r)))
			log.Warn("%s", string(debug.Stack()))
			return
		}
	}()

	count := 0
	count += goawayGroup("reader", compReader)
	count += goawayGroup("general", compGeneral)
	count += goawayGroup("writer", compWriter)
	count += goawayGroup("preinit", compPreInit)

	if count == 0 {
		log.Debug("there are no goaway component")
		return
	}
	log.Info("goaway %d component", count)
}

// goawayGroup calls Goaway() on every component in a single type group that
// implements FatimaRuntimeGoaway, concurrently, bounded by componentGroupTimeout.
// It returns the number of goaway targets in the group. A panic in one component
// is isolated and does not affect the others.
func goawayGroup(name string, list []fatima.FatimaComponent) int {
	target := make([]fatima.FatimaRuntimeGoaway, 0)
	for _, v := range list {
		if comp, ok := v.(fatima.FatimaRuntimeGoaway); ok {
			target = append(target, comp)
		}
	}
	if len(target) == 0 {
		return 0
	}

	done := make(chan struct{})
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(len(target))
		for _, v := range target {
			t := v
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						log.Warn("**PANIC** while goaway [%s] component : %s", name, r)
						log.Warn("%s", string(debug.Stack()))
					}
				}()
				t.Goaway()
			}()
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("goaway group [%s] %d component(s) done", name, len(target))
	case <-time.After(componentGroupTimeout):
		log.Warn("goaway group [%s] timeout after %v, proceeding to next group", name, componentGroupTimeout)
	}
	return len(target)
}
