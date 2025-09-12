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
 * @date 25. 9. 11. 오후 4:03
 */

package ipc

import (
	"fmt"
	"time"

	log "github.com/fatima-go/fatima-log"
)

const (
	junoProgramName = "juno"
)

// isApplicationJuno 현재 프로세스가 juno인지 아닌지 확인
func isApplicationJuno() bool {
	return envProvideHelper.getProgramName() == junoProgramName
}

func registerGoAwaySessionListener() {
	RegisterIPCSessionListener(newGoAwaySessionListener())
}

func newGoAwaySessionListener() FatimaIPCSessionListener {
	return &GoAwaySessionListener{}
}

type GoAwaySessionListener struct {
}

func (g *GoAwaySessionListener) StartSession(ctx SessionContext) {
	log.Trace("[%s] start session", ctx)
}

func (g *GoAwaySessionListener) OnClose(ctx SessionContext) {
	log.Trace("[%s] on close", ctx)
}

func (g *GoAwaySessionListener) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Trace("IPC command incoming : %s", message)

	if !message.Is(CommandGoaway) {
		return
	}

	defer ctx.Close()

	log.Warn("IPC process CommandGoaway : %s", message)
	transactionId := AsString(message.Data.GetValue(DataKeyTransaction))
	if len(transactionId) == 0 {
		log.Warn("[%s] received empty transaction id", ctx)
		return
	}

	if isApplicationJuno() {
		log.Trace("[%s] juno call itself", ctx)
		g.callGoaway(ctx, transactionId)
		return
	}

	err := g.validateTransaction(ctx, transactionId)
	if err != nil {
		log.Warn("[%s] fail to validate transaction : %s", ctx, err.Error())
	} else {
		g.callGoaway(ctx, transactionId)
	}
}

// validateTransaction goaway 요청이 들어온 transaction을 juno에게 질의해서 확인
func (g *GoAwaySessionListener) validateTransaction(ctx SessionContext, transactionId string) error {
	// STEP 1: ask transaction is valid to juno
	// prepare junoClient
	junoClient, err := newFatimaIPCClientSession(junoProgramName)
	if err != nil {
		return fmt.Errorf("cannot make connection to %s : %s", junoProgramName, err.Error())
	}
	defer junoClient.Disconnect()

	// send transaction verify command
	err = junoClient.SendCommand(NewMessageTransactionVerify(transactionId))
	if err != nil {
		return fmt.Errorf("fail to send transaction(%s) verify : %s", transactionId, err.Error())
	}

	c1 := make(chan Message, 1)
	go func() {
		// receive response from juno
		clientMessage, e1 := junoClient.ReadCommand()
		if e1 != nil {
			log.Warn("[%s] fail to read command : %s", ctx, e1.Error())
			return
		}
		c1 <- clientMessage
	}()

	// determine transaction from response is valid or not
	select {
	case junoResponse := <-c1:
		if !junoResponse.Is(CommandTransactionVerifyDone) {
			return fmt.Errorf("unexpected response from juno : %s", junoResponse)
		}
		receivedTransactionId := AsString(junoResponse.Data.GetValue(DataKeyTransaction))
		if transactionId != receivedTransactionId {
			return fmt.Errorf("transaction id mismatch : [%s:%s]", transactionId, receivedTransactionId)
		}
		verified := AsBool(junoResponse.Data.GetValue(DataKeyVerify))
		if !verified {
			return fmt.Errorf("transaction verify fail : [%s:%t]", transactionId, verified)
		}

		// STEP 2: proceed goaway if verified
		log.Trace("[%s] transaction verify success : %s", ctx, transactionId)
	case <-time.After(time.Second):
		return fmt.Errorf("timeout to receive transaction verify done : %s", transactionId)
	}
	return nil
}

func (g *GoAwaySessionListener) callGoaway(ctx SessionContext, transactionId string) {
	if goawayRunner == nil {
		return
	}

	if isApplicationJuno() {
		goawayRunner.Goaway()
		return
	}

	// send goaway start command
	err := ctx.SendCommand(NewMessageGoawayStart(transactionId))
	if err != nil {
		log.Warn("[%s] fail to send goaway start : %s, %s", ctx, transactionId, err.Error())
	} else {
		log.Warn("[%s] sent goaway start : %s", ctx, transactionId)
	}
	goawayRunner.Goaway()
	if err == nil {
		// send goaway done command
		err = ctx.SendCommand(NewMessageGoawayDone(transactionId))
		if err != nil {
			log.Warn("[%s] fail to send goaway done : %s, %s", ctx, transactionId, err.Error())
		} else {
			log.Warn("[%s] sent goaway done : %s", ctx, transactionId)
		}
	}
}
