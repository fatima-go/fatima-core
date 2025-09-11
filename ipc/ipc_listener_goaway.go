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
	"sync"
	"time"

	log "github.com/fatima-go/fatima-log"
)

const (
	junoProgramName = "juno"
)

func registerGoAwaySessionListener() {
	RegisterIPCSessionListener(newGoAwaySessionListener())
}

func newGoAwaySessionListener() FatimaIPCSessionListener {
	listener := &GoAwaySessionListener{}
	listener.clientSessionMap = make(map[string]clientSessionRecord)
	listener.cleanSessionTick = time.NewTicker(time.Minute)
	go func() {
		for range listener.cleanSessionTick.C {
			listener.clientSessionLock.Lock()
			for id, session := range listener.clientSessionMap {
				if session.isExpired() {
					delete(listener.clientSessionMap, id)
					log.Trace("[IPC] clientSession %s removed", id)
					continue
				}
			}
			listener.clientSessionLock.Unlock()
		}
	}()
	return listener
}

type clientSessionRecord struct {
	ctx   SessionContext
	epoch time.Time
}

func (c clientSessionRecord) isExpired() bool {
	return time.Now().Before(c.epoch.Add(time.Minute * 2))
}

type GoAwaySessionListener struct {
	clientSessionLock sync.Mutex
	clientSessionMap  map[string]clientSessionRecord
	cleanSessionTick  *time.Ticker
}

func (g *GoAwaySessionListener) addClientSession(ctx SessionContext) {
	g.clientSessionLock.Lock()
	defer g.clientSessionLock.Unlock()
	g.clientSessionMap[ctx.String()] = clientSessionRecord{ctx: ctx, epoch: time.Now()}
}

func (g *GoAwaySessionListener) removeClientSession(ctx SessionContext) {
	g.clientSessionLock.Lock()
	defer g.clientSessionLock.Unlock()
	delete(g.clientSessionMap, ctx.String())
}

func (g *GoAwaySessionListener) StartSession(ctx SessionContext) {
	log.Trace("[%s] start session", ctx)
	g.addClientSession(ctx)
}

func (g *GoAwaySessionListener) callGoaway(ctx SessionContext, transactionId string) {
	if goawayRunner == nil {
		return
	}

	err := ctx.SendCommand(NewMessageGoawayStart(transactionId))
	if err != nil {
		log.Warn("[%s] fail to send goaway start : %s, %s", ctx, transactionId, err.Error())
	}
	log.Trace("[%s] sent goaway start : %s", ctx, transactionId)
	goawayRunner.Goaway()
	if err == nil {
		err = ctx.SendCommand(NewMessageGoawayDone(transactionId))
		if err != nil {
			log.Warn("[%s] fail to send goaway done : %s, %s", ctx, transactionId, err.Error())
		} else {
			log.Trace("[%s] sent goaway done : %s", ctx, transactionId)
		}
	}
}

func (g *GoAwaySessionListener) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Trace("[%s] on receive command : %s", ctx, message)

	if !message.Is(CommandGoaway) {
		return
	}

	transactionId := AsString(message.Data.GetValue(DataKeyTransaction))
	if len(transactionId) == 0 {
		log.Warn("[%s] received empty transaction id", ctx)
		ctx.Close()
		return
	}

	if envProvideHelper.getProgramName() == junoProgramName {
		log.Trace("[%s] juno call itself", ctx)
		g.callGoaway(ctx, transactionId)
		return
	}

	// STEP 1 : ask transaction is valid to juno
	// prepare client
	junoClient, err := newFatimaIPCClientSession(junoProgramName)
	if err != nil {
		log.Warn("[%s] cannot make connection to %s : %s", ctx, junoProgramName, err.Error())
		ctx.Close()
		return
	}

	// send transaction verify command
	err = junoClient.SendCommand(NewMessageTransactionVerify(transactionId))
	if err != nil {
		log.Warn("[%s] fail to send transaction(%s) verify : %s", ctx, transactionId, err.Error())
		ctx.Close()
		return
	}

	c1 := make(chan Message, 1)
	go func() {
		// receive response from juno
		clientMessage, e1 := junoClient.ReadCommand()
		if e1 != nil {
			log.Warn("[%s] fail to read command : %s", ctx, e1.Error())
			ctx.Close()
			return
		}
		c1 <- clientMessage
	}()

	// determine transaction from response is valid or not
	select {
	case msg := <-c1:
		if !msg.Is(CommandTransactionVerifyDone) {
			log.Warn("[%s] unexpected message received : %s", ctx, msg)
			break
		}
		receivedTransactionId := AsString(message.Data.GetValue(DataKeyTransaction))
		if transactionId != receivedTransactionId {
			log.Warn("[%s] transaction id mismatch : %s - %s", ctx, transactionId, receivedTransactionId)
			break
		}
		verified := AsBool(msg.Data.GetValue(DataKeyVerify))
		if !verified {
			log.Warn("[%s] transaction verify fail : %s - %t", ctx, transactionId, verified)
			break
		}

		// STEP 2 : proceed goaway if verified
		log.Trace("[%s] transaction verify success : %s", ctx, transactionId)
		g.callGoaway(ctx, transactionId)
	case <-time.After(time.Second):
		log.Warn("[%s] timeout to receive transaction verify done : %s", ctx, transactionId)
	}

	// close sessions
	junoClient.Disconnect()
	ctx.Close()
}

func (g *GoAwaySessionListener) OnClose(ctx SessionContext) {
	log.Trace("[%s] on close", ctx)
	g.removeClientSession(ctx)
}
