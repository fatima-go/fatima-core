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
	NotifyFrom      = "go-fatima"
	NotifyInitiator = "go-fatima"
)

func buildAlarmMessage(fatimaRuntime fatima.FatimaRuntime, level monitor.AlarmLevel, action monitor.ActionType, message string, category string) []byte {
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

	content := make(map[string]interface{})
	var notifyType monitor.NotifyType
	notifyType = monitor.NotifyAlarm
	content["type"] = notifyType.String() // type : notify level
	if !action.IsNil() {
		content["action"] = action.String() // action : kind of action
	}
	content["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	content["alarm_level"] = level.String()
	content["from"] = NotifyFrom
	content["initiator"] = NotifyInitiator
	content["message"] = message
	if action.IsProcessStartup() {
		content["deployment"] = GetProcessDeployment()
	}

	if len(category) > 0 {
		content["category"] = category
	}

	body["message"] = content

	m["header"] = header
	m["body"] = body

	b, err := json.Marshal(m)
	if err != nil {
		log.Warn("fail to make content json : %s", err.Error())
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

	content := make(map[string]interface{})
	var notifyType monitor.NotifyType
	notifyType = monitor.NotifyEvent
	content["type"] = notifyType.String() // type : notify level
	content["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	content["from"] = NotifyFrom
	content["initiator"] = NotifyInitiator
	content["message"] = message

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
		content["params"] = args
	}

	body["message"] = content

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
