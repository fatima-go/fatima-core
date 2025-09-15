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
	"github.com/fatima-go/fatima-core/builder"
	"github.com/fatima-go/fatima-core/monitor"
	"github.com/fatima-go/fatima-log"
)

type SystemAwareManagement struct {
	runtimeProcess *builder.FatimaRuntimeProcess
	monitor        monitor.SystemStatusMonitor
	awareHA        []monitor.FatimaSystemHAAware
	awarePS        []monitor.FatimaSystemPSAware
}

func newSystemAwareManagement(runtimeProcess *builder.FatimaRuntimeProcess, mon monitor.SystemStatusMonitor) *SystemAwareManagement {
	instance := new(SystemAwareManagement)

	instance.runtimeProcess = runtimeProcess
	// awareHA observers
	instance.awareHA = make([]monitor.FatimaSystemHAAware, 0)
	// awarePS observers
	instance.awarePS = make([]monitor.FatimaSystemPSAware, 0)
	instance.monitor = mon
	currentStatus := runtimeProcess.GetSystemStatus().(*builder.FatimaPackageSystemStatus)

	ps, _ := mon.GetPSStatus()
	currentStatus.SetPSStatus(ps)
	ha, _ := mon.GetHAStatus()
	currentStatus.SetHAStatus(ha)

	return instance
}

func (s *SystemAwareManagement) RegisterSystemHAAware(aware monitor.FatimaSystemHAAware) {
	s.awareHA = append(s.awareHA, aware)
}

func (s *SystemAwareManagement) RegisterSystemPSAware(aware monitor.FatimaSystemPSAware) {
	s.awarePS = append(s.awarePS, aware)
}

func (s *SystemAwareManagement) SystemHAStatusChanged(newHAStatus monitor.HAStatus) {
	log.Warn("new HA Status detected : %s", newHAStatus)
	for _, aware := range s.awareHA {
		aware.SystemHAStatusChanged(newHAStatus)
	}
}

func (s *SystemAwareManagement) SystemPSStatusChanged(newPSStatus monitor.PSStatus) {
	log.Warn("new PS Status detected : %s", newPSStatus)
	for _, aware := range s.awarePS {
		aware.SystemPSStatusChanged(newPSStatus)
	}
}

func (s *SystemAwareManagement) Process() {
	currentStatus := s.runtimeProcess.GetSystemStatus().(*builder.FatimaPackageSystemStatus)

	// check and deliver PS change
	if ps, ok := s.monitor.GetPSStatus(); ok {
		oldps := currentStatus.GetPSStatus()
		if oldps != ps {
			currentStatus.SetPSStatus(ps)
			go func() {
				s.SystemPSStatusChanged(ps)
			}()
		}
	}

	// check and deliver HA change
	if ha, ok := s.monitor.GetHAStatus(); ok {
		oldha := currentStatus.GetHAStatus()
		if oldha != ha {
			currentStatus.SetHAStatus(ha)
			go func() {
				s.SystemHAStatusChanged(ha)
			}()
		}
	}

	// check and deliver loglevel change
	if logLevel, ok := s.monitor.GetLogLevel(); ok {
		if s.runtimeProcess.GetLogLevel() != logLevel {
			log.SetLevel(logLevel)
			log.Warn("fatima proc log level : %s", logLevel)
			s.runtimeProcess.SetLogLevel(logLevel)
		}
	}
}
