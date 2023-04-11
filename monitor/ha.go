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

const (
	HA_STATUS_UNKNOWN = 0
	HA_STATUS_ACTIVE  = 1
	HA_STATUS_STANDBY = 2
)

type HAStatus uint8

func (this HAStatus) String() string {
	switch this {
	case HA_STATUS_ACTIVE:
		return "Active"
	case HA_STATUS_STANDBY:
		return "Standby"
	}
	return "Unknown"
}

func ToHAStatus(value int) HAStatus {
	switch value {
	case HA_STATUS_ACTIVE:
		return HA_STATUS_ACTIVE
	case HA_STATUS_STANDBY:
		return HA_STATUS_STANDBY
	}
	return HA_STATUS_UNKNOWN
}
