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
 * @date 25. 9. 9. 오후 1:38
 *
 */

package ipc

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	log "github.com/fatima-go/fatima-log"
	"github.com/stretchr/testify/assert"
)

const (
	testProgramName = "test"
	testProgramPid  = 0
)

func init() {
	log.Initialize(log.NewPreference(""))
}

func mockGetProgramName() string {
	return testProgramName
}

func mockGetPid(proc string) (int, error) {
	return testProgramPid, nil
}

func mockGetSockDir() string {
	return "/tmp"
}

func mockBuildAddress() string {
	return filepath.Join(envProvideHelper.getSockDir(),
		fmt.Sprintf("%s%s.%d.sock",
			sockFilePrefix,
			envProvideHelper.getProgramName(),
			testProgramPid,
		),
	)
}

func beforeTestEnv() {
	envProvideHelper.getPid = mockGetPid
	envProvideHelper.getSockDir = mockGetSockDir
	envProvideHelper.buildAddress = mockBuildAddress
	envProvideHelper.getProgramName = mockGetProgramName
}

func TestEnv(t *testing.T) {
	beforeTestEnv()

	assert.EqualValues(t, "/tmp", envProvideHelper.getSockDir())

	listener := &TestSessionListener{}
	RegisterIPCSessionListener(listener)

	startIPCServer()

	// prepare client
	clientSession, err := newFatimaIPCClientSession(testProgramName)
	if !assert.Nil(t, err) {
		t.Fatalf("fail to create client session : %s", err.Error())
	}

	// send goaway
	err = clientSession.SendCommand(NewMessageGoaway())
	if !assert.Nil(t, err) {
		t.Fatalf("fail to send command : %s", err.Error())
	}

	// read transaction verify
	message, err := clientSession.ReadCommand()
	if !assert.Nil(t, err) {
		t.Fatalf("fail to read command : %s", err.Error())
	}
	assert.True(t, message.Is(CommandTransactionVerify))
	transaction := AsString(message.Data.GetValue(DataKeyTransaction))
	assert.True(t, len(transaction) > 0)

	// send transaction verify done
	err = clientSession.SendCommand(NewMessageTransactionVerifyDone(transaction, true))
	if !assert.Nil(t, err) {
		t.Fatalf("fail to send command : %s", err.Error())
	}
	time.Sleep(time.Second)
	clientSession.Disconnect()
	stopIPCServer()

	assert.True(t, listener.sessionStarted)
	assert.True(t, listener.receiveCommand)
	assert.True(t, listener.closeCalled)
	assert.True(t, listener.transactionMatched)
}

type TestSessionListener struct {
	sessionStarted     bool
	receiveCommand     bool
	closeCalled        bool
	transactionMatched bool
	transaction        string
}

func (t *TestSessionListener) StartSession(ctx SessionContext) {
	log.Info("start session : %s", ctx)
	t.sessionStarted = true
}

func (t *TestSessionListener) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Info("OnReceiveCommand : %s", ctx)
	log.Info("message : %s", message)
	t.receiveCommand = true

	if message.Is(CommandGoaway) {
		t.transaction = AsString(message.Data.GetValue(DataKeyTransaction))
		if len(t.transaction) == 0 {
			log.Warn("[%s] received empty transaction id", ctx)
			return
		}
		err := ctx.SendCommand(NewMessageTransactionVerify(t.transaction))
		if err != nil {
			log.Warn("fail to send transaction verify : %s", err.Error())
		}
		log.Debug("[%s] sent transaction verify : %s", ctx, t.transaction)
		return
	} else if message.Is(CommandTransactionVerifyDone) {
		transaction := AsString(message.Data.GetValue(DataKeyTransaction))
		if t.transaction == transaction {
			t.transactionMatched = true
		}
	}
}

func (t *TestSessionListener) OnClose(ctx SessionContext) {
	log.Info("OnClose : %s", ctx)
	t.closeCalled = true
}
