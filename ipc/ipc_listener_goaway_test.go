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
 * @author dave
 * @date 25. 9. 11. 오후 8:03
 */

package ipc

import (
	"net"
	"testing"
	"time"

	log "github.com/fatima-go/fatima-log"
	"github.com/stretchr/testify/assert"
)

func mockGetJunoProgramName() string {
	return junoProgramName
}

func beforeTestProviderForGoAwayListener() {
	envProvideHelper.getPid = mockGetPid
	envProvideHelper.getSockDir = mockGetSockDir
	envProvideHelper.getProgramName = mockGetJunoProgramName
	envProvideHelper.buildAddress = mockBuildAddress
	goawayRunner = &dummyGoawayRunner{}
}

func startSimulation(junoSimulator dummyListener, programNameProvider provideFunc) dummyListener {
	runJunoServer(junoSimulator)
	runUserApplicationServer(newMockSessionContext(junoSimulator), programNameProvider)
	return junoSimulator
}

// runJunoServer juno 시뮬레이터를 IPC 서버로 등록
func runJunoServer(junoSimulator FatimaIPCSessionListener) {
	beforeTestProviderForGoAwayListener()
	envProvideHelper.getProgramName = mockGetJunoProgramName
	RegisterIPCSessionListener(junoSimulator)
	startIPCServer()
}

// runUserApplicationServer 사용자 프로그램의 IPC listener 시작
func runUserApplicationServer(ctx SessionContext, programNameProvider provideFunc) {
	envProvideHelper.getProgramName = programNameProvider
	listener := newGoAwaySessionListener()
	listener.StartSession(ctx)
	listener.OnReceiveCommand(ctx, NewMessageGoaway())
}

// TestWithUserApplicationJuno juno 프로세스가 goaway 를 받았을 경우
// 스스로 goaway 를 호출하는 케이스는 없다
func TestWithUserApplicationJuno(t *testing.T) {
	junoSimulator := startSimulation(&dummyJunoSimulator{}, mockGetJunoProgramName)
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)
	// goaway listener는 goaway 를 받은 프로세스가 juno 스스로일 경우 별도의 action을 하지 않는다
	assert.False(t, junoSimulator.isSessionStarted())
	assert.False(t, junoSimulator.isReceiveCommand())
	assert.False(t, junoSimulator.isGoawayStartCalled())
	assert.False(t, junoSimulator.isGoawayDoneCalled())
}

// TestJunoVerifyMiss 사용자 프로세스가 goaway 를 받았을 경우
// juno에서 verify done 을 주지 않아서 timeout 나는 케이스
func TestJunoVerifyMiss(t *testing.T) {
	junoSimulator := startSimulation(&dummyJunoSimulator{}, mockGetProgramName)
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)
	assert.True(t, junoSimulator.isSessionStarted())
	assert.True(t, junoSimulator.isReceiveCommand())
	assert.True(t, junoSimulator.isCloseCalled())

	// transaction verify 가 성공하지 못하면 goaway start/done 을 처리하지 않는다
	assert.False(t, junoSimulator.isGoawayStartCalled())
	assert.False(t, junoSimulator.isGoawayDoneCalled())
}

// TestJunoVerifyFail 사용자 프로세스가 goaway 를 받았을 경우
// juno에서 verify done 을 false 로 전달
func TestJunoVerifyFail(t *testing.T) {
	junoSimulator := startSimulation(&verifyFalseJunoSimulator{}, mockGetProgramName)
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)
	assert.True(t, junoSimulator.isSessionStarted())
	assert.True(t, junoSimulator.isReceiveCommand())
	assert.True(t, junoSimulator.isCloseCalled())

	// transaction verify 가 성공하지 못하면 goaway start/done 을 처리하지 않는다
	assert.False(t, junoSimulator.isGoawayStartCalled())
	assert.False(t, junoSimulator.isGoawayDoneCalled())
}

// TestJunoVerifySuccess 사용자 프로세스가 goaway 를 받았을 경우
// juno에서 verify done 을 true 로 전달
func TestJunoVerifySuccess(t *testing.T) {
	junoSimulator := startSimulation(&verifyTrueJunoSimulator{}, mockGetProgramName)
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)
	assert.True(t, junoSimulator.isSessionStarted())
	assert.True(t, junoSimulator.isReceiveCommand())
	assert.True(t, junoSimulator.isCloseCalled())
	assert.True(t, junoSimulator.isGoawayStartCalled())
	assert.True(t, junoSimulator.isGoawayDoneCalled())
}

// TestJunoResponseInvalidTransaction 사용자 프로세스가 goaway 를 받았을 경우
// juno에서 요청된 트랜잭션이 아닌 다른 트랜잭션 값을 응답
func TestJunoResponseInvalidTransaction(t *testing.T) {
	junoSimulator := startSimulation(&invalidTransactionJunoSimulator{}, mockGetProgramName)
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)
	assert.True(t, junoSimulator.isSessionStarted())
	assert.True(t, junoSimulator.isReceiveCommand())
	assert.True(t, junoSimulator.isCloseCalled())
	assert.False(t, junoSimulator.isGoawayStartCalled())
	assert.False(t, junoSimulator.isGoawayDoneCalled())
}

func newMockSessionContext(junoSimulator FatimaIPCSessionListener) SessionContext {
	return &mockSessionContext{junoSimulator: junoSimulator}
}

type mockSessionContext struct {
	junoSimulator FatimaIPCSessionListener
}

func (m *mockSessionContext) String() string {
	return "mockSessionContext"
}

func (m *mockSessionContext) Close() {

}

func (m *mockSessionContext) GetConnection() net.Conn {
	return nil
}

func (m *mockSessionContext) SendCommand(message Message) error {
	if m.junoSimulator != nil {
		m.junoSimulator.OnReceiveCommand(m, message)
	}
	return nil
}

type dummyListener interface {
	FatimaIPCSessionListener
	isSessionStarted() bool
	isReceiveCommand() bool
	isCloseCalled() bool
	isGoawayStartCalled() bool
	isGoawayDoneCalled() bool
}

type dummyJunoSimulator struct {
	sessionStarted    bool
	receiveCommand    bool
	closeCalled       bool
	goawayStartCalled bool
	goawayDoneCalled  bool
}

func (t *dummyJunoSimulator) isSessionStarted() bool {
	return t.sessionStarted
}

func (t *dummyJunoSimulator) isReceiveCommand() bool {
	return t.receiveCommand
}

func (t *dummyJunoSimulator) isCloseCalled() bool {
	return t.closeCalled
}

func (t *dummyJunoSimulator) isGoawayStartCalled() bool {
	return t.goawayStartCalled
}

func (t *dummyJunoSimulator) isGoawayDoneCalled() bool {
	return t.goawayDoneCalled
}

func (t *dummyJunoSimulator) StartSession(ctx SessionContext) {
	log.Info("start session : %s", ctx)
	t.sessionStarted = true
}

func (t *dummyJunoSimulator) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Info("[sim] OnReceiveCommand : %s", message)
	t.receiveCommand = true
}

func (t *dummyJunoSimulator) OnClose(ctx SessionContext) {
	log.Info("OnClose : %s", ctx)
	t.closeCalled = true
}

type dummyGoawayRunner struct {
	calledGoaway bool
}

func (d *dummyGoawayRunner) Goaway() {
	d.calledGoaway = true
	log.Trace("called goaway")
}

type verifyFalseJunoSimulator struct {
	dummyJunoSimulator
}

func (t *verifyFalseJunoSimulator) StartSession(ctx SessionContext) {
	log.Info("start session : %s", ctx)
	t.sessionStarted = true
}

func (t *verifyFalseJunoSimulator) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Info("[sim] OnReceiveCommand : %s", message)
	t.receiveCommand = true

	if !message.Is(CommandTransactionVerify) {
		return
	}

	transactionId := AsString(message.Data.GetValue(DataKeyTransaction))
	if len(transactionId) == 0 {
		log.Warn("[%s] received empty transaction id", ctx)
		return
	}
	err := ctx.SendCommand(NewMessageTransactionVerifyDone(transactionId, false))
	if err != nil {
		log.Warn("fail to send transaction verify done : %s", err.Error())
	}
	log.Debug("[%s] sent transaction verify false : %s", ctx, transactionId)
}

func (t *verifyFalseJunoSimulator) OnClose(ctx SessionContext) {
	log.Info("OnClose : %s", ctx)
	t.closeCalled = true
}

type verifyTrueJunoSimulator struct {
	dummyJunoSimulator
}

func (t *verifyTrueJunoSimulator) StartSession(ctx SessionContext) {
	log.Info("start session : %s", ctx)
	t.sessionStarted = true
}

func (t *verifyTrueJunoSimulator) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Info("[sim] OnReceiveCommand : %s", message)
	t.receiveCommand = true

	if message.Is(CommandTransactionVerify) {
		transactionId := AsString(message.Data.GetValue(DataKeyTransaction))
		if len(transactionId) == 0 {
			log.Warn("[%s] received empty transaction id", ctx)
			return
		}
		err := ctx.SendCommand(NewMessageTransactionVerifyDone(transactionId, true))
		if err != nil {
			log.Warn("fail to send transaction verify done : %s", err.Error())
		}
		log.Debug("[%s] sent transaction verify true : %s", ctx, transactionId)
	} else if message.Is(CommandGoawayStart) {
		t.goawayStartCalled = true
	} else if message.Is(CommandGoawayDone) {
		t.goawayDoneCalled = true
	}
}

func (t *verifyTrueJunoSimulator) OnClose(ctx SessionContext) {
	log.Info("OnClose : %s", ctx)
	t.closeCalled = true
}

type invalidTransactionJunoSimulator struct {
	dummyJunoSimulator
}

func (t *invalidTransactionJunoSimulator) StartSession(ctx SessionContext) {
	log.Info("start session : %s", ctx)
	t.sessionStarted = true
}

func (t *invalidTransactionJunoSimulator) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Info("[sim] OnReceiveCommand : %s", message)
	t.receiveCommand = true

	if message.Is(CommandTransactionVerify) {
		transactionId := AsString(message.Data.GetValue(DataKeyTransaction))
		if len(transactionId) == 0 {
			log.Warn("[%s] received empty transaction id", ctx)
			return
		}
		transactionId = "another_random_transaction"
		err := ctx.SendCommand(NewMessageTransactionVerifyDone(transactionId, true))
		if err != nil {
			log.Warn("fail to send transaction verify done : %s", err.Error())
		}
		log.Debug("[%s] sent transaction verify true : %s", ctx, transactionId)
	} else if message.Is(CommandGoawayStart) {
		t.goawayStartCalled = true
	} else if message.Is(CommandGoawayDone) {
		t.goawayDoneCalled = true
	}
}

func (t *invalidTransactionJunoSimulator) OnClose(ctx SessionContext) {
	log.Info("OnClose : %s", ctx)
	t.closeCalled = true
}
