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
	"bytes"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-log"
	"gopkg.in/yaml.v3"
	"os"
	"sort"
	"strings"
)

type ProcessItem struct {
	Gid       int    `yaml:"gid"`
	Name      string `yaml:"name"`
	Loglevel  string `yaml:"loglevel"`
	Hb        bool   `yaml:"hb,omitempty"`
	Path      string `yaml:"path,omitempty"`
	Grep      string `yaml:"grep,omitempty"`
	Startmode int    `yaml:"startmode,omitempty"`
	Weight    int    `yaml:"weight,omitempty"`   // 프로세스 가동 weight. weight 값이 높은 프로세스들을 먼저 가동시킨다
	StartSec  int    `yaml:"startsec,omitempty"` // 프로세스가 가동 후 온라인이 될때까지 충분히 보장되는 시간(초)
}

func (p ProcessItem) GetGid() int {
	return p.Gid
}

func (p ProcessItem) GetName() string {
	return p.Name
}

func (p ProcessItem) GetHeartbeat() bool {
	return p.Hb
}

func (p ProcessItem) GetPath() string {
	return p.Path
}

func (p ProcessItem) GetGrep() string {
	return p.Grep
}

func (p ProcessItem) GetWeight() int {
	return p.Weight
}

func (p ProcessItem) GetStartSec() int {
	return p.StartSec
}

func (p ProcessItem) GetStartMode() fatima.ProcessStartMode {
	switch p.Startmode {
	case 1:
		return fatima.StartModeAlone
	case 2:
		return fatima.StartModeByHA
	case 3:
		return fatima.StartModeByPS
	default:
		return fatima.StartModeByJuno
	}
}

func (p ProcessItem) GetLogLevel() log.LogLevel {
	return buildLogLevel(p.Loglevel)
}

type GroupItem struct {
	Id   int    `yaml:"id"`
	Name string `yaml:"name"`
}

type GroupItems []GroupItem

func (a GroupItems) Len() int           { return len(a) }
func (a GroupItems) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a GroupItems) Less(i, j int) bool { return a[i].Id < a[j].Id }

type YamlFatimaPackageConfig struct {
	env        fatima.FatimaEnv
	predefines fatima.Predefines
	Groups     []GroupItem   `yaml:"group,flow"`
	Processes  []ProcessItem `yaml:"process"`
}

// NewYamlFatimaPackageConfig load fatima package processes information
// from $FATIMA_HOME/conf/fatima-package.yaml
func NewYamlFatimaPackageConfig(env fatima.FatimaEnv) *YamlFatimaPackageConfig {
	instance := new(YamlFatimaPackageConfig)
	instance.env = env
	instance.Reload()
	return instance
}

func (y *YamlFatimaPackageConfig) OrderByGroup() {
	ordered := make([]ProcessItem, 0)

	sort.Sort(GroupItems(y.Groups))

	// pickup OPM first
	for _, v := range y.Processes {
		if v.Gid == 1 {
			ordered = append(ordered, v)
		}
	}

	for i := 1; i < len(y.Groups); i++ {
		for _, v := range y.Processes {
			if v.Gid == y.Groups[i].Id {
				ordered = append(ordered, v)
			}
		}
	}

	y.Processes = ordered
}

func (y *YamlFatimaPackageConfig) Save() {
	d, err := yaml.Marshal(y)
	if err != nil {
		log.Warn("fail to create yaml data : %s", err.Error())
		return
	}

	var comment = "---\n" +
		"# this is fatima-package.yaml sample\n" +
		"# group (group id, group name)\n" +
		"# process list\n" +
		"# column : gid, name, loglevel, path, startmode\n" +
		"# - gid : group id\n" +
		"# - startmode : default 0(always started by juno), 1(not started by juno), 2(by HA), 3(by PS)\n"

	var buff bytes.Buffer
	buff.WriteString(comment)
	buff.Write(d)
	err = os.WriteFile(y.env.GetFolderGuide().GetPackageProcFile(), buff.Bytes(), 0644)
	if err != nil {
		log.Warn("fail to save yaml configuration file : %s", err.Error())
		return
	}
}

// Reload read file ($FATIMA_HOME/conf/fatima-package.yaml)
func (y *YamlFatimaPackageConfig) Reload() {
	data, err := os.ReadFile(y.env.GetFolderGuide().GetPackageProcFile())
	check(err)

	err = yaml.Unmarshal(data, &y)
	check(err)

	if len(y.Groups) == 0 || len(y.Processes) == 0 {
		panic(fmt.Errorf("invalid fatima yaml configuration : %s", y.env.GetFolderGuide().GetPackageProcFile()))
	}
}

func (y *YamlFatimaPackageConfig) GetProcByName(name string) fatima.FatimaPkgProc {
	for _, each := range y.Processes {
		if each.Name == name {
			return each
		}
	}

	return nil
}

func (y *YamlFatimaPackageConfig) GetProcByGroup(name string) []fatima.FatimaPkgProc {
	procList := make([]fatima.FatimaPkgProc, 0)
	gid := y.GetGroupId(name)
	if gid < 0 {
		return procList
	}

	for _, each := range y.Processes {
		if each.Gid == gid {
			procList = append(procList, each)
		}
	}

	return procList
}

func (y *YamlFatimaPackageConfig) GetAllProc(exceptOpmGroup bool) []fatima.FatimaPkgProc {
	procList := make([]fatima.FatimaPkgProc, 0)
	if !exceptOpmGroup {
		for _, v := range y.Processes {
			procList = append(procList, v)
		}
		return procList
	}

	gid := y.GetGroupId("OPM")
	for _, each := range y.Processes {
		if each.Gid != gid {
			procList = append(procList, each)
		}
	}

	return procList
}

func (y *YamlFatimaPackageConfig) GetGroupId(groupName string) int {
	comp := strings.ToLower(groupName)
	for _, each := range y.Groups {
		if comp == strings.ToLower(each.Name) {
			return each.Id
		}
	}

	return -1
}

func (y *YamlFatimaPackageConfig) IsValidGroupId(groupId int) bool {
	for _, each := range y.Groups {
		if each.Id == groupId {
			return true
		}
	}

	return false
}

func buildLogLevel(s string) log.LogLevel {
	switch strings.ToLower(s) {
	case "trace":
		return log.LOG_TRACE
	case "debug":
		return log.LOG_DEBUG
	case "info":
		return log.LOG_INFO
	case "warn":
		return log.LOG_WARN
	case "error":
		return log.LOG_ERROR
	case "none":
		return log.LOG_NONE
	}
	return log.LOG_TRACE
}

type DummyFatimaPackageConfig struct {
	env        fatima.FatimaEnv
	predefines fatima.Predefines
	Groups     []GroupItem   `yaml:"group,flow"`
	Processes  []ProcessItem `yaml:"process"`
}

func NewDummyFatimaPackageConfig(env fatima.FatimaEnv) *DummyFatimaPackageConfig {
	instance := new(DummyFatimaPackageConfig)
	instance.env = env
	instance.Reload()
	return instance
}

func (y *DummyFatimaPackageConfig) Reload() {
}

func (y *DummyFatimaPackageConfig) GetProcByName(name string) fatima.FatimaPkgProc {
	item := ProcessItem{}
	item.Name = name
	item.Startmode = fatima.StartModeAlone
	item.Path = "/"
	item.Gid = 0
	item.Hb = false
	item.Loglevel = "debug"
	return item
}
