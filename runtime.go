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

package fatima

import (
	"github.com/fatima-go/fatima-core/monitor"
)

type FatimaRuntimeInteractor interface {
	Register(component FatimaComponent)
	RegisterSystemHAAware(aware monitor.FatimaSystemHAAware)
	RegisterSystemPSAware(aware monitor.FatimaSystemPSAware)
	RegisterMeasureUnit(unit monitor.SystemMeasurable)
	Run()
	Stop()
}

type FatimaRuntimeGoaway interface {
	Goaway()
}

type ProcessInteractor interface {
	FatimaRuntimeInteractor
	FatimaRuntimeGoaway
	Initialize() bool
	Shutdown()
}

type FatimaRuntime interface {
	GetEnv() FatimaEnv
	GetConfig() Config
	GetPackaging() Packaging
	GetSystemStatus() monitor.FatimaSystemStatus
	GetSystemNotifyHandler() monitor.SystemNotifyHandler
	IsRunning() bool
	FatimaRuntimeInteractor
}
