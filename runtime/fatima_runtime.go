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

package runtime

import (
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/builder"
	"github.com/fatima-go/fatima-core/infra"
)

var process *builder.FatimaRuntimeProcess

// GetFatimaRuntime return fatima runtime
func GetFatimaRuntime() fatima.FatimaRuntime {
	return GetGeneralFatimaRuntime()
}

// GetGeneralFatimaRuntime return general(typical process type) purpose fatima runtime
func GetGeneralFatimaRuntime() fatima.FatimaRuntime {
	if process != nil {
		return process
	}

	// prepare process
	process = builder.NewFatimaRuntime()

	// set runtime builder which has fatima package process information and config
	builder := getRuntimeBuilder(process.GetEnv(), fatima.PROCESS_TYPE_GENERAL)

	// initializing process using runtime builder
	process.Initialize(builder)

	// set interactor
	process.SetInteractor(infra.NewProcessInteractor(process))

	return process
}

// GetUserInteractiveFatimaRuntime return fatima runtime for user interactive type process
func GetUserInteractiveFatimaRuntime(controller interface{}) fatima.FatimaRuntime {
	if process != nil {
		return process
	}

	// prepare process
	process = builder.NewFatimaRuntime()

	// set builder
	builder := getRuntimeBuilder(process.GetEnv(), fatima.PROCESS_TYPE_UI)
	process.Initialize(builder)

	// set interactor
	process.SetInteractor(infra.NewProcessInteractor(process))

	// regist user interactive
	process.Regist(newUserInteractive(controller))

	return process
}
