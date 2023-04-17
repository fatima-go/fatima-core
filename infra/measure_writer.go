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
	"bytes"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	SUFFIX_MONITOR = "monitor"
	FOLDER_MONITOR = "monitor"
	FOLDER_HISTORY = "history"
	MAX_FILE_SIZE  = 30 << (10 * 2) // 30 MB
)

type MeasurementWriter interface {
	write(measurement)
}

func newMeasureFileWriter(env fatima.FatimaEnv) *MeasureFileWriter {
	instance := new(MeasureFileWriter)
	fileName := fmt.Sprintf("%s.%d.%s",
		env.GetSystemProc().GetProgramName(),
		env.GetSystemProc().GetPid(),
		SUFFIX_MONITOR)
	baseDir := filepath.Join(env.GetFolderGuide().GetAppProcFolder(), FOLDER_MONITOR)
	instance.filePath = filepath.Join(baseDir, fileName)
	instance.historyPath = filepath.Join(baseDir, FOLDER_HISTORY)
	ensureDirectory(baseDir, true)
	ensureDirectory(instance.historyPath, true)

	except := fmt.Sprintf("%s.%d", env.GetSystemProc().GetProgramName(), env.GetSystemProc().GetPid())
	moveOldToHistory(env.GetSystemProc().GetProgramName(), baseDir, instance.historyPath, except)

	go func() {
		clearOldHistory(instance.historyPath)
	}()

	return instance
}

// clearOldHistory delete old(before 24 hour) monitor files from history folder
func clearOldHistory(path string) {
	// find files in log path
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}

	diff := 24 * time.Hour
	deadline := time.Now().Add(-diff)
	for _, file := range files {
		if file.ModTime().After(deadline) {
			continue
		}
		os.Remove(filepath.Join(path, file.Name()))
	}
}

// moveOldToHistory move old monitor record files into history folder
func moveOldToHistory(procName string, folder string, history string, except string) {
	files, _ := filepath.Glob(fmt.Sprintf("%s%c%s.*", folder, filepath.Separator, procName))
	for _, v := range files {
		if isDirectory(v) {
			continue
		}
		filename := filepath.Base(v)
		if strings.HasPrefix(filename, except) {
			continue
		}
		newpath := filepath.Join(history, filename)
		os.Rename(v, newpath)
	}
}

type MeasureFileWriter struct {
	filePath    string
	historyPath string
}

func (this *MeasureFileWriter) write(msr measurement) {
	checkSwitch(this.filePath)

	filePtr, err := os.OpenFile(this.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer filePtr.Close()

	var buffer bytes.Buffer
	// [2017/02/11 00:33:23]
	buffer.WriteString("-------------------------------------------------------------------")
	buffer.WriteByte('\n')
	buffer.WriteByte('\n')
	buffer.WriteString(fmt.Sprintf("[%s]\n", msr.eventTime.Format("2006-01-02 15:04:05")))
	for _, v := range msr.items {
		buffer.WriteString(fmt.Sprintf("[%s]\n", v.keyName))
		buffer.WriteString(v.value)
		buffer.WriteByte('\n')
	}
	filePtr.WriteString(buffer.String())
}

func checkSwitch(active string) {
	stat, err := os.Stat(active)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	if stat.Size() > MAX_FILE_SIZE {
		log.Trace("switch monitor file...")
		backup := fmt.Sprintf("%s.backup", active)
		os.Rename(active, backup)
		return
	}
}

/*
[2017/02/11 00:33:23]
[org.fatima.core.process.FatimaProcess]
:: gc=0,gc_time=0,total_gc=0,heap_used=16809760,heap_commit=25165824,heap_max=1073741824,direct_used=2367
[RabbitMQ-Consumer-ThreadPool]
:: queue=0000, complete=0000, active=0000, pool=0000, largestpool=0000, core=0004
[StandaloneDSImpl]
:: active=00/08, idle=00/08
[AMPQ]
:: Deliver/Ack/Reject =
[RestClient]
:: lease=000, pending=000, available=000, max=010 [total]
:: lease=000, pending=000, available=000, max=004 [http://172.21.85.73:8080]



*/
