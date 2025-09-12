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

/*
Field name   | Mandatory? | Allowed values  | Allowed special characters
----------   | ---------- | --------------  | --------------------------
Seconds      | Yes        | 0-59            | * / , -
Minutes      | Yes        | 0-59            | * / , -
Hours        | Yes        | 0-23            | * / , -
Day of month | Yes        | 1-31            | * / , - ?
Month        | Yes        | 1-12 or JAN-DEC | * / , -
Day of week  | Yes        | 0-6 or SUN-SAT  | * / , - ?

c.AddFunc("0 30 * * * *", func() { fmt.Println("Every hour on the half hour") })
c.AddFunc("@hourly",      func() { fmt.Println("Every hour") })
c.AddFunc("@every 1h30m", func() { fmt.Println("Every hour thirty") })

Entry                  | Description                                | Equivalent To
-----                  | -----------                                | -------------
@yearly (or @annually) | Run once a year, midnight, Jan. 1st        | 0 0 0 1 1 *
@monthly               | Run once a month, midnight, first of month | 0 0 0 1 * *
@weekly                | Run once a week, midnight on Sunday        | 0 0 0 * * 0
@daily (or @midnight)  | Run once a day, midnight                   | 0 0 0 * * *
@hourly                | Run once an hour, beginning of hour        | 0 0 * * * *
*/

package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-log"
	robfig_cron "github.com/robfig/cron"
)

const (
	fileRerun             = "cron.rerun"
	configPrefix          = "cron."
	configSuffixSpec      = ".spec"
	configSuffixDesc      = ".desc"
	configSuffixPrimary   = ".primary"
	configSuffixProfile   = ".profile"
	configSuffixSample    = ".sample"
	configSuffixRunUnique = ".rununique"
)

var (
	cronCreationLock      sync.RWMutex
	cron                  *robfig_cron.Cron
	cronJobList           []*CronJob = make([]*CronJob, 0)
	fatimaRuntime         fatima.FatimaRuntime
	oneSecondTick         *time.Ticker
	lastRerunModifiedTime time.Time
	jobRunningMutex       = sync.Mutex{}
	runningCronJobs       = make(map[string]struct{})

	errInvalidConfig = errors.New("invalid fatima config")
)

type CronJob struct {
	name      string
	desc      string
	spec      string
	args      []string
	sample    string
	runUnique bool
	primary   bool
	profile   string
	runnable  func(string, fatima.FatimaRuntime, ...string)
}

func (c CronJob) Run() {
	if !c.canRunnable() {
		return
	}

	log.Info("start job [%s]", c.name)
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic to execute : %s", r)
			log.Error("%s", string(debug.Stack()))
		}
		jobRunningMutex.Lock()
		defer jobRunningMutex.Unlock()
		delete(runningCronJobs, c.name)
	}()

	startMillis := CurrentTimeMillis()
	c.runnable(c.desc, fatimaRuntime, c.args...)
	endMillis := CurrentTimeMillis()

	log.Info("cron job [%s] elapsed %d milli seconds", c.name, endMillis-startMillis)
}

func (c CronJob) canRunnable() bool {
	if !fatimaRuntime.IsRunning() {
		return false
	}

	if !c.isProfileMatch() {
		log.Info("cron job [%s] skipped because profile is not match", c.name)
		return false
	}

	if c.primary {
		if !fatimaRuntime.GetSystemStatus().IsPrimary() {
			log.Info("cron job [%s] skipped because system is not PRIMARY", c.name)
			return false
		}
	}
	if !c.runUnique {
		return true
	}

	jobRunningMutex.Lock()
	defer jobRunningMutex.Unlock()
	_, ok := runningCronJobs[c.name]
	if ok {
		log.Warn("job %s is running", c.name)
		return false
	}
	runningCronJobs[c.name] = struct{}{}
	return true
}

func (c CronJob) isProfileMatch() bool {
	if len(c.profile) == 0 {
		return true
	}

	comp := strings.ToUpper(fatimaRuntime.GetEnv().GetProfile())
	for _, v := range strings.Split(c.profile, ",") {
		token := strings.TrimSpace(v)
		if strings.ToUpper(token) == comp {
			return true
		}
	}

	log.Info("mismatch profile : [%s=%s]", fatimaRuntime.GetEnv().GetProfile(), c.profile)
	return false
}

func StartCron() {
	if cron == nil {
		return
	}

	if len(cron.Entries()) == 0 {
		return
	}

	registerCronjobCommandsToJuno()

	log.Info("total %d cron jobs scheduled", len(cron.Entries()))
	cron.Start()
}

func StopCron() {
	if cron == nil {
		return
	}

	cron.Stop()
	if oneSecondTick != nil {
		oneSecondTick.Stop()
		oneSecondTick = nil
		log.Info("cron jobs stopped")
	}
}

func registerCronjobCommandsToJuno() {
	if len(cronJobList) == 0 {
		return
	}

	processCommand := make(map[string]interface{})
	processCommand["process"] = fatimaRuntime.GetEnv().GetSystemProc().GetProgramName()
	cronCommands := make([]interface{}, 0)

	for _, job := range cronJobList {
		command := make(map[string]string)
		command["name"] = job.name
		command["desc"] = job.desc
		command["spec"] = job.spec
		command["sample"] = job.sample
		cronCommands = append(cronCommands, command)
	}
	processCommand["jobs"] = cronCommands
	b, _ := json.Marshal(processCommand)

	dir := filepath.Join(fatimaRuntime.GetEnv().GetFolderGuide().GetFatimaHome(),
		"data",
		"juno",
		"crons")

	err := ensureDirectory(dir)
	if err != nil {
		log.Warn("fail to register cron commands to juno : %s", err.Error())
		return
	}

	file := filepath.Join(dir, fatimaRuntime.GetEnv().GetSystemProc().GetProgramName()+".json")
	err = os.WriteFile(file, b, 0644)
	if err != nil {
		log.Warn("fail to write cron commands to juno : %s", err.Error())
		return
	}
}

func ensureDirectory(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0755)
		}
	}

	return nil
}

func Rerun(jobName string, args []string) {
	log.Info("try to rerun job [%s]", jobName)
	for _, job := range cronJobList {
		if job.name == jobName {
			go func() {
				job.args = args
				job.Run()
				job.args = nil
			}()
			return
		}
	}
}

func RegisterCronJob(runtime fatima.FatimaRuntime, jobName string, runnable func(string, fatima.FatimaRuntime, ...string)) error {
	if runtime.GetConfig() == nil {
		return errInvalidConfig
	}

	ensureSingleCronInstance(runtime)

	job, err := newCronJob(runtime.GetConfig(), jobName, runnable)
	if err != nil {
		return err
	}

	err = cron.AddJob(job.spec, job)
	if err != nil {
		return err
	}

	log.Info("job[%s] scheduled : %s", jobName, job.spec)
	cronJobList = append(cronJobList, job)

	return nil
}

func ensureSingleCronInstance(runtime fatima.FatimaRuntime) {
	cronCreationLock.Lock()
	if cron == nil {
		cron = robfig_cron.New()
		fatimaRuntime = runtime
		clearRerunFile()
		startRerunFileScanner()
	}
	cronCreationLock.Unlock()
}

func newCronJob(config fatima.Config, name string, runnable func(string, fatima.FatimaRuntime, ...string)) (*CronJob, error) {
	key := fmt.Sprintf("%s%s%s", configPrefix, name, configSuffixSpec)
	spec, ok := config.GetValue(key)
	if !ok {
		return nil, errors.New("insufficient config key " + key)
	}

	key = fmt.Sprintf("%s%s%s", configPrefix, name, configSuffixDesc)
	desc, ok := config.GetValue(key)
	if !ok {
		desc = name
	}

	key = fmt.Sprintf("%s%s%s", configPrefix, name, configSuffixProfile)
	profile, ok := config.GetValue(key)

	key = fmt.Sprintf("%s%s%s", configPrefix, name, configSuffixPrimary)
	primary, ok := config.GetValue(key)

	key = fmt.Sprintf("%s%s%s", configPrefix, name, configSuffixSample)
	sample, ok := config.GetValue(key)

	key = fmt.Sprintf("%s%s%s", configPrefix, name, configSuffixRunUnique)
	unique, err := config.GetBool(key)
	if err != nil {
		unique = true
	}

	job := &CronJob{}
	job.name = name
	job.desc = desc
	job.spec = spec
	job.profile = profile
	job.primary = true
	if len(primary) > 0 && strings.ToLower(primary) != "true" {
		job.primary = false
	}
	job.sample = strings.TrimSpace(sample)
	job.runnable = runnable
	job.runUnique = unique
	return job, nil
}

func clearRerunFile() {
	file := filepath.Join(fatimaRuntime.GetEnv().GetFolderGuide().GetDataFolder(), fileRerun)
	_ = os.Remove(file)
}

func startRerunFileScanner() {
	oneSecondTick = time.NewTicker(time.Second * 1)
	go func() {
		for range oneSecondTick.C {
			scanRerunFile()
		}
	}()
}

func scanRerunFile() {
	file := filepath.Join(fatimaRuntime.GetEnv().GetFolderGuide().GetDataFolder(), fileRerun)
	stat, err := os.Stat(file)
	if err != nil {
		return
	}

	if lastRerunModifiedTime == stat.ModTime() {
		return
	}

	lastRerunModifiedTime = stat.ModTime()
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}

	jobNameAndArgs := strings.Trim(string(data), "\r\n ")
	if len(jobNameAndArgs) > 0 {
		command := strings.Split(jobNameAndArgs, " ")
		jobName := command[0]
		var jobArgs []string
		if len(command) > 1 {
			jobArgs = command[1:]
		}
		Rerun(jobName, jobArgs)
		clearRerunFile()
	}
}
