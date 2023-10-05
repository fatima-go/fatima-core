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

package monitor

import (
	"fmt"
)

type SystemNotifyHandler interface {
	SendAlarm(level AlarmLevel, action ActionType, message string)
	SendAlarmWithCategory(level AlarmLevel, action ActionType, message string, category string)
	SendActivity(json interface{})
	SendEvent(message string, v ...interface{})
}

const (
	NotifyAlarm = iota
	NotifyEvent
)

type NotifyType uint8

func (n NotifyType) String() string {
	switch n {
	case NotifyAlarm:
		return "ALARM"
	case NotifyEvent:
		return "EVENT"
	}
	return fmt.Sprintf("Unknown notify value : %d", n)
}

const (
	AlarmLevelWarn = iota
	AlarmLevelMinor
	AlamLevelMajor
)

type AlarmLevel uint8

func (al AlarmLevel) String() string {
	switch al {
	case AlarmLevelWarn:
		return "WARN"
	case AlarmLevelMinor:
		return "MINOR"
	case AlamLevelMajor:
		return "MAJOR"
	}
	return fmt.Sprintf("Unknown alarm level value : %d", al)
}

const (
	ActionUnknown         = 0
	ActionProcessShutdown = 1
	ActionProcessStartup  = 2
)

type ActionType uint8

func (n ActionType) String() string {
	switch n {
	case ActionProcessShutdown:
		return "PROCESS_SHUTDOWN"
	case ActionProcessStartup:
		return "PROCESS_STARTUP"
	}
	return fmt.Sprintf("Unknown action value : %d", n)
}

func (n ActionType) IsNil() bool {
	switch n {
	case ActionProcessShutdown, ActionProcessStartup:
		return false
	}
	return true
}

func (n ActionType) IsProcessStartup() bool {
	switch n {
	case ActionProcessStartup:
		return true
	}
	return false
}
