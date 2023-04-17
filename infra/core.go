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
	"fmt"
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/builder"
	"github.com/fatima-go/fatima-core/lib"
	"github.com/fatima-go/fatima-core/monitor"
	"github.com/fatima-go/fatima-log"
	"net/http"
	"time"
)

type ProcessCoreWorker interface {
	Process()
}

var oneSecondTickWorkers []ProcessCoreWorker
var fiveSecondTickWorkers []ProcessCoreWorker

func init() {
	oneSecondTickWorkers = make([]ProcessCoreWorker, 0)
	fiveSecondTickWorkers = make([]ProcessCoreWorker, 0)
}

type DefaultProcessInteractor struct {
	runtimeProcess *builder.FatimaRuntimeProcess
	awareManager   *SystemAwareManagement
	monitor        monitor.SystemStatusMonitor
	measurement    *SystemMeasureManagement
	readers        []fatima.FatimaIOReader
}

func NewProcessInteractor(runtimeProcess *builder.FatimaRuntimeProcess) *DefaultProcessInteractor {
	instance := new(DefaultProcessInteractor)
	instance.runtimeProcess = runtimeProcess
	// monitor : Active/Standby, Primary/Secondary
	instance.monitor = newCentralFilebaseManagement(runtimeProcess.GetEnv())
	instance.awareManager = newSystemAwareManagement(runtimeProcess, instance.monitor)
	// special type of FatimaComponent. usually we need create 'Reader' type first
	instance.readers = make([]fatima.FatimaIOReader, 0)
	instance.measurement = newSystemMeasureManagement(runtimeProcess)

	// check HA/PS status every 1 second
	oneSecondTickWorkers = append(oneSecondTickWorkers, instance.awareManager)
	// process mgmt every 5 seconds
	fiveSecondTickWorkers = append(fiveSecondTickWorkers, instance.measurement)

	startTickers()
	return instance
}

// Regist regist FatimaComponent
func (i *DefaultProcessInteractor) Regist(component fatima.FatimaComponent) {
	registComponent(component)

	if comp, ok := component.(monitor.FatimaSystemHAAware); ok {
		i.RegistSystemHAAware(comp)
	}

	if comp, ok := component.(monitor.FatimaSystemPSAware); ok {
		i.RegistSystemPSAware(comp)
	}

	if comp, ok := component.(fatima.FatimaIOReader); ok {
		i.readers = append(i.readers, comp)
	}
}

func (i *DefaultProcessInteractor) RegistSystemHAAware(aware monitor.FatimaSystemHAAware) {
	i.awareManager.RegistSystemHAAware(aware)
}

func (i *DefaultProcessInteractor) RegistSystemPSAware(aware monitor.FatimaSystemPSAware) {
	i.awareManager.RegistSystemPSAware(aware)
}

func (i *DefaultProcessInteractor) Initialize() bool {
	return initializeComponent()
}

func (i *DefaultProcessInteractor) Goaway() {
	goawayComponent()
}

func (i *DefaultProcessInteractor) startListening() {
	for _, v := range i.readers {
		t := v
		go func() {
			t.StartListening()
		}()
	}
}

// RUN start process business activity
func (i *DefaultProcessInteractor) Run() {
	// start listening (Reader type FatimaComponent)
	i.startListening()

	// start batche jobs
	lib.StartCron()

	// notify process bootup
	bootupNotify()

	// start pprof service if relative property exists
	i.pprofService()
	if i.runtimeProcess.GetBuilder().GetProcessType() == fatima.PROCESS_TYPE_GENERAL {
		message := fmt.Sprintf("%s process started", i.runtimeProcess.GetEnv().GetSystemProc().GetProgramName())
		i.runtimeProcess.GetSystemNotifyHandler().SendAlarm(monitor.AlarmLevelMinor, message)
	}
}

func (i *DefaultProcessInteractor) Stop() {

}

func (i *DefaultProcessInteractor) Shutdown() {
	if i.runtimeProcess.GetBuilder().GetProcessType() == fatima.PROCESS_TYPE_GENERAL {
		message := fmt.Sprintf("%s process shutdowned", i.runtimeProcess.GetEnv().GetSystemProc().GetProgramName())
		i.runtimeProcess.GetSystemNotifyHandler().SendAlarm(monitor.AlamLevelMajor, message)
	}
	lib.StopCron()
	shutdownComponent(i.runtimeProcess.GetEnv().GetSystemProc().GetProgramName())
}

func (i *DefaultProcessInteractor) RegistMeasureUnit(unit monitor.SystemMeasurable) {
	i.measurement.registUnit(unit)
}

func (i *DefaultProcessInteractor) pprofService() {
	addr, ok := i.runtimeProcess.GetConfig().GetValue(builder.GOFATIMA_PROP_PPROF_ADDRESS)
	if ok {
		go func() {
			err := http.ListenAndServe(addr, http.DefaultServeMux)
			if err != nil {
				log.Warn("fail to start pprof service : {}", err.Error())
			}
		}()
		log.Info("starting pprof service. address=%s", addr)
	}
}

func startTickers() {
	oneSecondTick := time.NewTicker(time.Second * 1)
	go func() {
		for range oneSecondTick.C {
			iterateWorkers(oneSecondTickWorkers)
		}
	}()
	fiveSecondTick := time.NewTicker(time.Second * 5)
	go func() {
		for range fiveSecondTick.C {
			iterateWorkers(fiveSecondTickWorkers)
		}
	}()
}

func iterateWorkers(workers []ProcessCoreWorker) {
	if workers == nil || len(workers) < 1 {
		return
	}

	for _, v := range workers {
		v.Process()
	}
}
