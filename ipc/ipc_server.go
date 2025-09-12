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
 * @date 25. 9. 9. 오전 9:40
 *
 */

package ipc

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"

	log "github.com/fatima-go/fatima-log"
)

const (
	sockFilePrefix           = "fatima."
	ipcNetwork               = "unix"
	errCloseConnectionString = "use of closed network connection"
)

var ipcServerSocket net.Listener

func isServerRunning() bool {
	return ipcServerSocket != nil
}

func startIPCServer() {
	if ipcServerSocket != nil {
		return
	}

	log.Debug("start ipc listen")
	removeUnusedSockFiles()

	address := envProvideHelper.buildAddress()
	log.Debug("using ipc address : %s", address)
	var err error
	ipcServerSocket, err = net.Listen(ipcNetwork, address)
	if err != nil {
		log.Error("fail to listen on socket : %s", err.Error())
	}

	go serverReceiveLoop(address)
}

func serverReceiveLoop(address string) {
	for {
		log.Trace("IPC Waiting for connection...")
		connectedSocket, err := ipcServerSocket.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), errCloseConnectionString) {
				log.Error("fail to accept socket : %s", err.Error())
			}
			break
		}
		go startSession(newSessionContext(connectedSocket))
	}

	log.Debug("removing ipc socket file : %s", address)
	_ = os.Remove(address)
}

func startSession(ctx SessionContext) {
	log.Debug("[%s] new ipc session started", ctx)

	propagateSessionStarted(ctx)

	scanner := bufio.NewScanner(ctx.GetConnection())
	for scanner.Scan() {
		d := scanner.Bytes()
		message, err := parseMessage(d)
		if err != nil {
			log.Warn("[%s] fail to parse initiator : %s", ctx, err.Error())
			continue
		}
		propagateOnReceiveCommand(ctx, message)
	}
	if err := scanner.Err(); err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			log.Warn("[%s] read timeout : %s", ctx, err.Error())
		} else if !errors.Is(io.EOF, err) &&
			!strings.Contains(err.Error(), "use of closed network connection") {
			log.Warn("[%s] fail to read socket : %s", ctx, err.Error())
		}
	}

	log.Debug("[%s] client disconnected", ctx)
	ctx.Close()
	propagateOnClose(ctx)
}

func stopIPCServer() {
	log.Debug("stop ipc listen")
	if ipcServerSocket != nil {
		_ = ipcServerSocket.Close()
		ipcServerSocket = nil
	}
}

func removeUnusedSockFiles() {
	dir := envProvideHelper.getSockDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Warn("fail to read dir : %s", err.Error())
		return
	}

	removeList := make([]string, 0)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), sockFilePrefix) {
			continue
		}
		info, _ := e.Info()
		if info.Mode()&fs.ModeSocket != 0 {
			removeList = append(removeList, filepath.Join(dir, e.Name()))
		}
	}

	for _, e := range removeList {
		log.Debug("ipc socket file removed : %s", e)
		_ = os.Remove(e)
	}
}
