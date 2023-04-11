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
	"github.com/fatima-go/fatima-core/monitor"
)

type FatimaPackageSystemStatus struct {
	psstatus monitor.PSStatus
	hastatus monitor.HAStatus
}

func (this *FatimaPackageSystemStatus) GetPSStatus() monitor.PSStatus {
	return this.psstatus
}

func (this *FatimaPackageSystemStatus) GetHAStatus() monitor.HAStatus {
	return this.hastatus
}

func (this *FatimaPackageSystemStatus) SetPSStatus(status monitor.PSStatus) {
	this.psstatus = status
}

func (this *FatimaPackageSystemStatus) SetHAStatus(status monitor.HAStatus) {
	this.hastatus = status
}

func (this *FatimaPackageSystemStatus) IsActive() bool {
	return this.hastatus == monitor.HA_STATUS_ACTIVE
}

func (this *FatimaPackageSystemStatus) IsPrimary() bool {
	return this.psstatus == monitor.PS_STATUS_PRIMARY
}
