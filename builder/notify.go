//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with p work for additional information
// regarding copyright ownership.  The ASF licenses p file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use p file except in compliance
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

package builder

import (
	"encoding/json"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/lib"
	"github.com/fatima-go/fatima-core/monitor"
	"github.com/fatima-go/fatima-log"
	"time"
)

const (
	ApplicationCode = 0x1
	LogicMeasure    = 10
	LogicNotify     = 20
)

func buildAlarmMessage(fatimaRuntime fatima.FatimaRuntime, level monitor.AlarmLevel, message string, category string) []byte {
	m := make(map[string]interface{})
	header := make(map[string]interface{})
	body := make(map[string]interface{})

	header["application_code"] = ApplicationCode
	header["logic"] = LogicNotify

	body["package_host"] = fatimaRuntime.GetPackaging().GetHost()
	body["package_name"] = fatimaRuntime.GetPackaging().GetName()
	body["package_group"] = fatimaRuntime.GetPackaging().GetGroup()
	body["package_profile"] = fatimaRuntime.GetEnv().GetProfile()
	body["package_process"] = fatimaRuntime.GetEnv().GetSystemProc().GetProgramName()
	body["event_time"] = lib.CurrentTimeMillis()

	alarm := make(map[string]interface{})
	//alarm["type"] = "ALARM"
	var notifyType monitor.NotifyType
	notifyType = monitor.NotifyAlarm
	alarm["type"] = notifyType.String()
	alarm["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	alarm["alarm_level"] = level.String()
	alarm["from"] = "go-fatima"
	alarm["initiator"] = "go-fatima"
	alarm["message"] = message

	if len(category) > 0 {
		alarm["category"] = category
	}

	body["message"] = alarm

	m["header"] = header
	m["body"] = body

	b, err := json.Marshal(m)
	if err != nil {
		log.Warn("fail to make alarm json : %s", err.Error())
		return nil
	}

	return b
}

func buildEventMessage(fatimaRuntime fatima.FatimaRuntime, message string, v ...interface{}) []byte {
	m := make(map[string]interface{})
	header := make(map[string]interface{})
	body := make(map[string]interface{})

	header["application_code"] = ApplicationCode
	header["logic"] = LogicNotify

	body["package_host"] = fatimaRuntime.GetPackaging().GetHost()
	body["package_name"] = fatimaRuntime.GetPackaging().GetName()
	body["package_group"] = fatimaRuntime.GetPackaging().GetGroup()
	body["package_profile"] = fatimaRuntime.GetEnv().GetProfile()
	body["package_process"] = fatimaRuntime.GetEnv().GetSystemProc().GetProgramName()
	body["event_time"] = lib.CurrentTimeMillis()

	alarm := make(map[string]interface{})
	//alarm["type"] = "EVENT"
	var notifyType monitor.NotifyType
	notifyType = monitor.NotifyEvent
	alarm["type"] = notifyType.String()
	alarm["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	alarm["from"] = "go-fatima"
	alarm["initiator"] = "go-fatima"
	alarm["message"] = message

	if len(v) > 0 {
		args := make([]string, 0)
		for _, a := range v {
			if e, ok := a.(fmt.Stringer); ok {
				args = append(args, e.String())
			} else if e, ok := a.(string); ok {
				args = append(args, e)
			} else if e, ok := a.(int); ok {
				args = append(args, fmt.Sprintf("%d", e))
			} else if e, ok := a.(float32); ok {
				args = append(args, fmt.Sprintf("%f", e))
			} else if e, ok := a.(float64); ok {
				args = append(args, fmt.Sprintf("%f", e))
			} else if e, ok := a.(int32); ok {
				args = append(args, fmt.Sprintf("%d", e))
			} else if e, ok := a.(uint32); ok {
				args = append(args, fmt.Sprintf("%d", e))
			} else if e, ok := a.(uint64); ok {
				args = append(args, fmt.Sprintf("%d", e))
			} else if e, ok := a.(bool); ok {
				args = append(args, fmt.Sprintf("%b", e))
			} else {
				args = append(args, ".")
			}
		}
		alarm["params"] = args
	}

	body["message"] = alarm

	m["header"] = header
	m["body"] = body

	b, err := json.Marshal(m)
	if err != nil {
		log.Warn("fail to make event json : %s", err.Error())
		return nil
	}

	return b
}

func buildActivityMessage(fatimaRuntime fatima.FatimaRuntime, v interface{}) []byte {
	m := make(map[string]interface{})
	header := make(map[string]interface{})
	body := make(map[string]interface{})

	header["application_code"] = ApplicationCode
	header["logic"] = LogicMeasure

	body["package_host"] = fatimaRuntime.GetPackaging().GetHost()
	body["package_name"] = fatimaRuntime.GetPackaging().GetName()
	body["package_group"] = fatimaRuntime.GetPackaging().GetGroup()
	body["package_profile"] = fatimaRuntime.GetEnv().GetProfile()
	body["package_process"] = fatimaRuntime.GetEnv().GetSystemProc().GetProgramName()
	body["event_time"] = lib.CurrentTimeMillis()
	body["message"] = v

	m["header"] = header
	m["body"] = body

	b, err := json.Marshal(m)
	if err != nil {
		log.Warn("fail to make alarm json : %s", err.Error())
		return nil
	}

	return b
}
