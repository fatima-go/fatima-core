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

package runtime

import (
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/builder"
	"github.com/fatima-go/fatima-core/monitor"
)

type DefaultProcessBuilder struct {
	pkgProcConfig fatima.FatimaPkgProcConfig
	predefines    fatima.Predefines
	config        fatima.Config
	monitor       monitor.SystemStatusMonitor
	systemAware   monitor.FatimaSystemAware
	processType   fatima.FatimaProcessType
}

func (this *DefaultProcessBuilder) GetPkgProcConfig() fatima.FatimaPkgProcConfig {
	return this.pkgProcConfig
}

func (this *DefaultProcessBuilder) GetPredefines() fatima.Predefines {
	return this.predefines
}

func (this *DefaultProcessBuilder) GetConfig() fatima.Config {
	return this.config
}

func (this *DefaultProcessBuilder) GetProcessType() fatima.FatimaProcessType {
	return this.processType
}

func (this *DefaultProcessBuilder) GetSystemStatusMonitor() monitor.SystemStatusMonitor {
	return this.monitor
}

func (this *DefaultProcessBuilder) GetSystemAware() monitor.FatimaSystemAware {
	return this.systemAware
}

// getRuntimeBuilder return fatima runtime builder
func getRuntimeBuilder(env fatima.FatimaEnv, processType fatima.FatimaProcessType) builder.FatimaRuntimeBuilder {
	processBuilder := new(DefaultProcessBuilder)
	processBuilder.processType = processType
	if processType == fatima.PROCESS_TYPE_GENERAL {
		processBuilder.pkgProcConfig = builder.NewYamlFatimaPackageConfig(env)
	} else {
		// USER INTERACTIVE
		processBuilder.pkgProcConfig = builder.NewDummyFatimaPackageConfig(env)
	}

	// predefines : serving fatima package global(shared) properties
	processBuilder.predefines = builder.NewPropertyPredefineReader(env)

	// load configuration
	processBuilder.config = builder.NewPropertyConfigReader(env, processBuilder.predefines)

	return processBuilder
}
