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
 * @date 25. 9. 9. 오전 10:56
 *
 */

package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	log "github.com/fatima-go/fatima-log"
)

func NewFatimaIPCClientSession(proc string) (FatimaIPCClientSession, error) {
	address, err := buildClientAddress(proc)
	if err != nil {
		return nil, fmt.Errorf("fail to build client address : %s", err.Error())
	}

	conn, err := net.Dial(ipcNetwork, address)
	if err != nil {
		return nil, fmt.Errorf("fail to connect to socket : %s", err.Error())
	}

	clientSession := &defaultClientSession{}
	clientSession.messageChan = make(chan Message, 16)
	clientSession.ctx = newClientSessionContext(conn)
	clientSession.connected = true
	log.Debug("[%s] connection established. start reading", clientSession.ctx)
	go clientSession.startRead() // start read goroutine
	return clientSession, nil
}

func newClientSessionContext(conn net.Conn) SessionContext {
	return &defaultSessionContext{
		sessionType:   sessionTypeClient,
		conn:          conn,
		transactionId: time.Now().UnixMilli(),
	}
}

type defaultClientSession struct {
	ctx         SessionContext
	messageChan chan Message
	connected   bool
}

func (d *defaultClientSession) String() string {
	return d.ctx.String()
}

func (d *defaultClientSession) SendCommand(message Message) error {
	if d.connected == false {
		return nil
	}

	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("[%s] fail to marshal json : %s", d.ctx, err.Error())
	}

	payload := make([]byte, len(b)+1)
	copy(payload, b)
	payload[len(b)] = '\n'
	_, err = d.ctx.GetConnection().Write(payload)
	if err != nil {
		return fmt.Errorf("[%s] fail to write to socket : %s", d.ctx, err.Error())
	}
	if log.IsDebugEnabled() {
		log.Debug("[%s] send command : %s", d.ctx, string(b))
	}
	return nil
}

func (d *defaultClientSession) ReadCommand() (Message, error) {
	if d.connected == false {
		return Message{}, fmt.Errorf("[%s] not connected", d.ctx)
	}

	return <-d.messageChan, nil
}

func (d *defaultClientSession) startRead() {
	if d.connected == false {
		return
	}

	scanner := bufio.NewScanner(d.ctx.GetConnection())
	for scanner.Scan() {
		b := scanner.Bytes()
		message, err := parseMessage(b)
		if err != nil {
			log.Warn("[%s] fail to parse initiator : %s", d.ctx, err.Error())
			continue
		}
		if !d.connected {
			break
		}
		d.messageChan <- message
		log.Trace("[%s] recv from peer : %s", d.ctx, message)
	}

	if err := scanner.Err(); err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			log.Warn("[%s] read timeout : %s", d.ctx, err.Error())
		} else if !errors.Is(io.EOF, err) &&
			!strings.Contains(err.Error(), "use of closed network connection") {
			log.Warn("[%s] fail to read socket : %s", d.ctx, err.Error())
		}
	}
}

func (d *defaultClientSession) Disconnect() {
	if d.connected == false {
		return
	}
	log.Debug("[%s] disconnecting", d.ctx)
	d.connected = false
	d.ctx.Close()
	close(d.messageChan)
}

func buildClientAddress(proc string) (string, error) {
	pid, err := envProvideHelper.getPid(proc)
	if err != nil {
		return "", err
	}

	if !checkProcessRunning(proc, pid) {
		return "", fmt.Errorf("process not running")
	}

	return filepath.Join(
		buildSockDir(proc),
		fmt.Sprintf("%s%s.%d.sock",
			sockFilePrefix,
			proc,
			pid),
	), nil
}
