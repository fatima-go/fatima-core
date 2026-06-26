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
	"github.com/fatima-go/fatima-core/builder"
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

// componentGroupTimeout is the fallback group timeout used when config is not
// injected (e.g. tests or direct calls). It bounds how long a single lifecycle
// group (goaway or shutdown) may take before the orchestrator stops waiting and
// proceeds to the next group, preventing a hung component from blocking the whole
// sequence. The effective value is normally resolved from config via
// resolveGroupTimeout (gofatima.component.lifecycle.timeout.seconds) and may be
// overridden per component through fatima.FatimaComponentLifecycleTimeout.
var componentGroupTimeout = 30 * time.Second

// resolveGroupTimeout reads the base (config-default) lifecycle group timeout from
// config. It returns componentGroupTimeout (30s) when the key is missing, unparsable
// or non-positive. This is the timeout applied to a group when no member component
// declares its own via fatima.FatimaComponentLifecycleTimeout.
func resolveGroupTimeout(config fatima.Config) time.Duration {
	if config == nil {
		return componentGroupTimeout
	}
	sec, err := config.GetInt(builder.GofatimaComponentLifecycleTimeoutSeconds)
	if err != nil || sec <= 0 {
		return componentGroupTimeout
	}
	return time.Duration(sec) * time.Second
}

// effectiveGroupTimeout computes the group's effective timeout as the maximum of
// each member's effective timeout. A member declaring fatima.FatimaComponentLifecycleTimeout
// with a positive value uses that value; otherwise it uses base (the config default).
// Since members in a group run concurrently, waiting for the longest member lets the
// whole group finish. An empty group falls back to base.
//
// Note base is the default for members that do NOT declare, not a global floor: a
// group whose members all declare values below base waits only for the longest of
// those declarations (per CR R2/R4). If any member is undeclared it contributes base,
// so base still applies in the common mixed case.
func effectiveGroupTimeout(components []fatima.FatimaComponent, base time.Duration) time.Duration {
	if len(components) == 0 {
		return base
	}
	var effective time.Duration
	for _, c := range components {
		if d := componentTimeout(c, base); d > effective {
			effective = d
		}
	}
	return effective
}

// componentTimeout returns the effective timeout a single component requests. It
// honors fatima.FatimaComponentLifecycleTimeout when implemented with a positive
// value, otherwise falls back to base.
func componentTimeout(c interface{}, base time.Duration) time.Duration {
	if lt, ok := c.(fatima.FatimaComponentLifecycleTimeout); ok {
		if d := lt.LifecycleTimeout(); d > 0 {
			return d
		}
	}
	return base
}

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
// is bounded by its effective timeout: the maximum of base (the config default,
// gofatima.component.lifecycle.timeout.seconds) and any per-component override
// declared via fatima.FatimaComponentLifecycleTimeout. Note this ordering only
// yields a true graceful shutdown when READER components honor the contract
// "stop accepting -> wait in-flight -> return" in their Shutdown().
func shutdownComponent(program string, base time.Duration) {
	log.Info("start shutdown FatimaComponent")

	defer func() {
		if r := recover(); r != nil {
			log.Error("**PANIC** while shutdown", errors.New(fmt.Sprintf("%s", r)))
			log.Error("%s", string(debug.Stack()))
			log.Close()
			return
		}
	}()

	shutdownGroup("reader", compReader, base)
	shutdownGroup("general", compGeneral, base)
	shutdownGroup("writer", compWriter, base)
	shutdownGroup("preinit", compPreInit, base)

	log.Warn("shutdown %s", program)
	log.Close()
}

// shutdownGroup shuts down every component in a single type group concurrently and
// waits up to the group's effective timeout for completion. The effective timeout is
// max(base, per-component overrides). A panic in one component is isolated and does
// not affect the others.
func shutdownGroup(name string, list []fatima.FatimaComponent, base time.Duration) {
	if len(list) == 0 {
		return
	}

	timeout := effectiveGroupTimeout(list, base)

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
	case <-time.After(timeout):
		log.Warn("shutdown group [%s] timeout after %v, proceeding to next group", name, timeout)
	}
}

// goawayComponent broadcasts Goaway() to components that implement
// FatimaRuntimeGoaway, in the same data-flow group order as shutdown
// (READER -> GENERAL -> WRITER -> PRE_INIT). Members within a group run
// concurrently; groups are sequential and bounded by the group's effective timeout
// (max of base and per-component overrides). Type does not restrict participation:
// any component implementing the interface is signaled, it only controls the order.
func goawayComponent(base time.Duration) {
	log.Info("start calling goaway...")

	defer func() {
		if r := recover(); r != nil {
			log.Warn("**PANIC** while goaway", errors.New(fmt.Sprintf("%s", r)))
			log.Warn("%s", string(debug.Stack()))
			return
		}
	}()

	count := 0
	count += goawayGroup("reader", compReader, base)
	count += goawayGroup("general", compGeneral, base)
	count += goawayGroup("writer", compWriter, base)
	count += goawayGroup("preinit", compPreInit, base)

	if count == 0 {
		log.Debug("there are no goaway component")
		return
	}
	log.Info("goaway %d component", count)
}

// goawayGroup calls Goaway() on every component in a single type group that
// implements FatimaRuntimeGoaway, concurrently, bounded by the group's effective
// timeout. The effective timeout is max(base, per-component overrides) computed over
// the goaway target set only (components not implementing FatimaRuntimeGoaway, and
// thus not signaled, do not contribute their declared timeout). It returns the number
// of goaway targets in the group. A panic in one component is isolated and does not
// affect the others.
func goawayGroup(name string, list []fatima.FatimaComponent, base time.Duration) int {
	// Collect goaway targets and compute the group's effective timeout over that
	// target set only, in a single pass. Components that do not implement
	// FatimaRuntimeGoaway are not signaled and must not contribute their declared
	// timeout (CR R4). target is non-empty past the guard below, so timeout is the
	// max of the targets' effective timeouts (base for any undeclared target).
	target := make([]fatima.FatimaRuntimeGoaway, 0)
	var timeout time.Duration
	for _, v := range list {
		if comp, ok := v.(fatima.FatimaRuntimeGoaway); ok {
			target = append(target, comp)
			if d := componentTimeout(comp, base); d > timeout {
				timeout = d
			}
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
	case <-time.After(timeout):
		log.Warn("goaway group [%s] timeout after %v, proceeding to next group", name, timeout)
	}
	return len(target)
}
