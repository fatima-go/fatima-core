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

package fatima

import (
	"github.com/fatima-go/fatima-log"
)

// log levels
const (
	StartModeByJuno = 0
	StartModeAlone  = 1
	StartModeByHA   = 2
	StartModeByPS   = 3
)

type ProcessStartMode int

type FatimaPkgProc interface {
	GetGid() int
	GetName() string
	GetHeartbeat() bool
	GetPath() string
	GetGrep() string
	GetLogLevel() log.LogLevel
	GetStartMode() ProcessStartMode
}

type FatimaPkgProcConfig interface {
	Reload()
	GetProcByName(name string) FatimaPkgProc
}
