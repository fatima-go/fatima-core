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

package monitor

import (
	"fmt"
)

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

type SystemNotifyHandler interface {
	SendAlarm(level AlarmLevel, message string)
	SendAlarmWithCategory(level AlarmLevel, message string, category string)
	SendActivity(json interface{})
	SendEvent(message string, v ...interface{})
}
