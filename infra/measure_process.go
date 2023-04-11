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
	"fmt"
	"runtime"
	"time"
)

func newProcessMeasurement() *ProcessMeasurement {
	measure := new(ProcessMeasurement)

	return measure
}

type ProcessMeasurement struct {
}

func (p *ProcessMeasurement) GetKeyName() string {
	return "fatima process"
}

var lastGcCount uint32

func (p *ProcessMeasurement) GetMeasure() string {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	if mem.LastGC == 0 {
		return fmt.Sprintf(" :: Alloc=%s, TotalAlloc=%s, Sys=%s, NumGC=%d",
			expressBytes(mem.Alloc),
			expressBytes(mem.TotalAlloc),
			expressBytes(mem.Sys),
			mem.NumGC)
	}

	pauseMs := 0
	if lastGcCount != mem.NumGC {
		pauseMs = int(mem.PauseNs[(mem.NumGC+255)%256] / 1000)
		lastGcCount = mem.NumGC
	}

	return fmt.Sprintf(" :: Alloc=%s, Sys=%s, TotalGC=%d, LastPause=%dms, LastGC=%s",
		expressBytes(mem.Alloc),
		//expressBytes(mem.TotalAlloc),
		expressBytes(mem.Sys),
		mem.NumGC,
		pauseMs,
		time.Unix(0, int64(mem.LastGC)).Format("2006-01-02 15:04:05"))
}
