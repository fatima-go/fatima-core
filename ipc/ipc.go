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
	"io"
	"io/fs"
	"os"

	"github.com/fatima-go/fatima-core"
	log "github.com/fatima-go/fatima-log"
)

type cronRunnableFunc func(string, []string)

func StartIPCService(fr fatima.FatimaRuntime,
	ps fatima.PlatformSupport,
	goawayImpl fatima.FatimaRuntimeGoaway,
	cronRunner cronRunnableFunc) io.Closer {
	if isServerRunning() {
		return nil
	}

	setFacilities(fr, ps, goawayImpl, cronRunner)

	// register a connection manager
	registerConnectionManager()
	// register goaway session listener
	registerGoAwaySessionListener()
	// register cron listener
	registerCronListener()

	// start server
	startIPCServer()

	return &ipcServiceCloser{}
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

type ipcServiceCloser struct{}

func (c *ipcServiceCloser) Close() error {
	stopIPCServer()
	return nil
}

func StopIPCService() {
	stopIPCServer()
}

var fatimaRuntime fatima.FatimaRuntime
var platformSupporter fatima.PlatformSupport
var goawayRunner fatima.FatimaRuntimeGoaway
var cronRunner cronRunnableFunc

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

func IsFatimaIPCAvailable(proc string) bool {
	if fatimaRuntime == nil {
		return false
	}

	pid, err := getPid(proc)
	if err != nil {
		return false
	}

	if !checkProcessRunning(proc, pid) {
		log.Debug("process %s [%d] is not running", proc, pid)
		return false
	}

	sockFile := buildAddressForProcess(
		buildSockDir(proc),
		proc,
		pid)
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
