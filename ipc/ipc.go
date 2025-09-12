/*
 * Copyright 2025 github.com/fatima-go
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
 * @author dave_01
 * @date 25. 9. 8. 오후 4:50
 *
 */

package ipc

import (
	"fmt"
	"io/fs"
	"os"
	"sync"

	"github.com/fatima-go/fatima-core"
	log "github.com/fatima-go/fatima-log"
)

type cronRunnableFunc func(string, []string)

var facilitiesSetOnce sync.Once

func StartIPCService(fr fatima.FatimaRuntime,
	ps fatima.PlatformSupport,
	goawayImpl fatima.FatimaRuntimeGoaway,
	cronRunner cronRunnableFunc) {
	facilitiesSetOnce.Do(func() {
		setFacilities(fr, ps, goawayImpl, cronRunner)
	})

	// register a connection manager
	registerConnectionManager()
	// register goaway session listener
	registerGoAwaySessionListener()
	// register cron listener
	registerCronListener()

	// start server
	startIPCServer()
}

func setFacilities(fr fatima.FatimaRuntime,
	ps fatima.PlatformSupport,
	goawayImpl fatima.FatimaRuntimeGoaway,
	runnable cronRunnableFunc) {
	fatimaRuntime = fr
	platformSupporter = ps
	goawayRunner = goawayImpl
	cronRunner = runnable
}

func StopIPCService() {
	stopIPCServer()
}

var fatimaRuntime fatima.FatimaRuntime
var platformSupporter fatima.PlatformSupport
var goawayRunner fatima.FatimaRuntimeGoaway
var cronRunner cronRunnableFunc

var ipcSessionListeners = make([]FatimaIPCSessionListener, 0)
var ipcSessionListenerLock sync.Mutex

func RegisterIPCSessionListener(listener FatimaIPCSessionListener) {
	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	ipcSessionListeners = append(ipcSessionListeners, listener)
}

type FatimaIPCSessionListener interface {
	StartSession(ctx SessionContext)
	OnReceiveCommand(ctx SessionContext, message Message)
	OnClose(ctx SessionContext)
}

type FatimaIPCClientSession interface {
	SendCommand(Message) error
	ReadCommand() (Message, error)
	Disconnect()
}

func IsSupportFatimaIPC(proc string) bool {
	if fatimaRuntime == nil {
		return false
	}

	pid, err := getPid(proc)
	if err != nil {
		return false
	}

	if !checkProcessRunning(proc, pid) {
		log.Warn("process %s [%d] is not running", proc, pid)
		return false
	}

	sockDir := fmt.Sprintf("%s/app/%s/data",
		fatimaRuntime.GetEnv().GetFolderGuide().GetFatimaHome(),
		proc,
	)
	sockFile := buildAddressForProcess(sockDir, proc, pid)
	stat, err := os.Stat(sockFile)
	if err != nil {
		return false
	}
	return stat.Mode()&fs.ModeSocket != 0
}

func checkProcessRunning(proc string, pid int) bool {
	if platformSupporter == nil {
		return true
	}
	return platformSupporter.CheckProcessRunningByPid(proc, pid)
}
