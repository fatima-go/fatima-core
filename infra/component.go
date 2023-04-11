//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// @project fatima
// @author DeockJin Chung (jin.freestyle@gmail.com)
// @date 2017. 3. 6. PM 7:42
//

package infra

import (
	"errors"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/lib"
	"github.com/fatima-go/fatima-log"
	"runtime/debug"
	"sync"
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

func registComponent(comp fatima.FatimaComponent) {
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

// initializeComponent : initialize registed FatimaComponent
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

func shutdownComponent(program string) {
	log.Info("start shutdown FatimaComponent")
	all := make([]fatima.FatimaComponent, 0)
	all = append(all, compPreInit...)
	all = append(all, compGeneral...)
	all = append(all, compReader...)
	all = append(all, compWriter...)

	defer func() {
		if r := recover(); r != nil {
			log.Error("**PANIC** while shutdown", errors.New(fmt.Sprintf("%s", r)))
			log.Error("%s", string(debug.Stack()))
			log.Close()
			return
		}
	}()

	size := len(all)
	if size > 0 {
		cyBarrier := lib.NewCyclicBarrier(size, func() {
			log.Warn("shutdown %s", program)
		})
		for _, v := range all {
			t := v
			cyBarrier.Dispatch(func() { t.Shutdown() })
		}
		cyBarrier.Wait()
	} else {
		log.Warn("shutdown %s", program)
	}

	log.Close()
}

func goawayComponent() {
	log.Info("start calling goaway...")
	defer func() {
		if r := recover(); r != nil {
			log.Warn("**PANIC** while initializing", errors.New(fmt.Sprintf("%s", r)))
			log.Warn("%s", string(debug.Stack()))
			return
		}
		//res = true
	}()

	all := make([]fatima.FatimaComponent, 0)
	all = append(all, compWriter...)
	all = append(all, compReader...)
	all = append(all, compGeneral...)
	all = append(all, compPreInit...)

	target := make([]fatima.FatimaRuntimeGoaway, 0)
	for _, v := range all {
		if comp, ok := v.(fatima.FatimaRuntimeGoaway); ok {
			target = append(target, comp)
		}
	}

	if len(target) == 0 {
		log.Debug("there are no goaway component")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(target))
	for _, v := range target {
		go callGoaway(&wg, v)
	}

	wg.Wait()
	log.Info("goaway %d component", len(target))
}

func callGoaway(wg *sync.WaitGroup, comp fatima.FatimaRuntimeGoaway) {
	defer wg.Done()
	comp.Goaway()
}
