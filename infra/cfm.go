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

package infra

import (
	"encoding/json"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/monitor"
	"github.com/fatima-go/fatima-log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type logLevelItem struct {
	process string
	level   string
}

type CentralFilebaseManagement struct {
	env fatima.FatimaEnv
}

func newCentralFilebaseManagement(env fatima.FatimaEnv) *CentralFilebaseManagement {
	instance := new(CentralFilebaseManagement)
	instance.env = env
	return instance
}

func (c *CentralFilebaseManagement) GetPSStatus() (monitor.PSStatus, bool) {
	filePath := filepath.Join(
		c.env.GetFolderGuide().GetFatimaHome(),
		"package",
		"cfm",
		"ha",
		"system.ps")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ioutil.WriteFile(filePath, []byte(strconv.Itoa(monitor.PS_STATUS_SECONDARY)), 0644)
			return monitor.PS_STATUS_SECONDARY, true
		}
		return monitor.PS_STATUS_UNKNOWN, true
	}
	value, err1 := strconv.Atoi(strings.Trim(string(data), "\r\n"))
	if err1 != nil {
		return monitor.PS_STATUS_UNKNOWN, false
	}
	return monitor.ToPSStatus(value), true
}

func (c *CentralFilebaseManagement) GetHAStatus() (monitor.HAStatus, bool) {
	filePath := filepath.Join(
		c.env.GetFolderGuide().GetFatimaHome(),
		"package",
		"cfm",
		"ha",
		"system.ha")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ioutil.WriteFile(filePath, []byte(strconv.Itoa(monitor.HA_STATUS_STANDBY)), 0644)
			return monitor.HA_STATUS_STANDBY, true
		}
		return monitor.HA_STATUS_UNKNOWN, true
	}
	value, err1 := strconv.Atoi(strings.Trim(string(data), "\r\n"))
	if err1 != nil {
		return monitor.HA_STATUS_UNKNOWN, false
	}
	return monitor.ToHAStatus(value), true
}

func (c *CentralFilebaseManagement) GetLogLevel() (log.LogLevel, bool) {
	filePath := filepath.Join(
		c.env.GetFolderGuide().GetFatimaHome(),
		"package",
		"cfm",
		"loglevels")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("read file fail", err)
		return log.LOG_NONE, false
	}

	//	var items []logLevelItem
	var items map[string]string
	err = json.Unmarshal(data, &items)
	if err != nil {
		log.Warn("fail to unmarshal loglevel to json : %s", err.Error())
		return log.LOG_NONE, false
	}

	value, ok := items[c.env.GetSystemProc().GetProgramName()]
	if !ok {
		// not found
		return log.LOG_NONE, false
	}

	loglevel, err1 := log.ConvertHexaToLogLevel(value)
	if err1 != nil {
		return log.LOG_NONE, false
	}

	return loglevel, true
}
