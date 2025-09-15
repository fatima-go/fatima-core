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
 * @date 25. 9. 9. 오전 10:42
 *
 */

package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

const (
	sessionTypeServer = 0
	sessionTypeClient = 1
)

type SessionType uint8

var ipcTransactionId int64

func newSessionContext(conn net.Conn) SessionContext {
	return &defaultSessionContext{
		sessionType:   sessionTypeServer,
		conn:          conn,
		transactionId: atomic.AddInt64(&ipcTransactionId, 1),
	}
}

type SessionContext interface {
	String() string
	Close()
	GetConnection() net.Conn
	SendCommand(message Message) error
}

type defaultSessionContext struct {
	sessionType   SessionType
	conn          net.Conn
	transactionId int64
	connLock      sync.Mutex
}

func (s *defaultSessionContext) GetConnection() net.Conn {
	return s.conn
}

func (s *defaultSessionContext) String() string {
	if s.sessionType == sessionTypeServer {
		return fmt.Sprintf("[S:%d]", s.transactionId)
	}
	return fmt.Sprintf("[C:%d]", s.transactionId)
}

func (s *defaultSessionContext) SendCommand(message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %s", err.Error())
	}

	payload := make([]byte, len(data)+1)
	copy(payload, data)
	payload[len(data)] = '\n'

	s.connLock.Lock()
	defer s.connLock.Unlock()
	if s.conn == nil {
		return fmt.Errorf("connection is not available")
	}
	_, err = s.conn.Write(payload)
	return err
}

func (s *defaultSessionContext) Close() {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}
