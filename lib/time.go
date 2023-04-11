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
// @date 2017. 3. 24. PM 11:27
//

package lib

import (
	"fmt"
	"time"
)

func CurrentTimeMillis() int {
	return int(time.Now().UnixNano() / 1000000)
}

func ExpressDuration(startMillis int) string {
	elapsed := CurrentTimeMillis() - startMillis
	if elapsed <= 0 {
		return "0 sec"
	}

	if elapsed > 60*1000 {
		// min..
		minute := elapsed / (60 * 1000)
		remain := elapsed % (60 * 1000)
		return fmt.Sprintf("%d min %d sec", minute, remain/1000)
	}
	return fmt.Sprintf("%d sec", elapsed/1000)
}
