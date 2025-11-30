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
 * @date 25. 9. 12. 오전 10:30
 */

package ipc

import (
	"sync"
	"time"

	log "github.com/fatima-go/fatima-log"
)

const (
	cleanSessionTickDuration = time.Minute
	sessionExpireDuration    = time.Minute * 2
)

func registerConnectionManager() {
	RegisterIPCSessionListener(newConnectionListener())
}

func newConnectionListener() FatimaIPCSessionListener {
	listener := &ConnectionListener{}
	listener.clientSessionMap = make(map[string]clientSessionRecord)
	listener.cleanSessionTick = time.NewTicker(cleanSessionTickDuration)
	go func() {
		for range listener.cleanSessionTick.C {
			listener.clientSessionLock.Lock()
			for id, session := range listener.clientSessionMap {
				if session.isExpired() {
					session.ctx.Close() // close force
					delete(listener.clientSessionMap, id)
					log.Warn("[IPC] clientSession %s removed", id)
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
	return time.Now().After(c.epoch.Add(sessionExpireDuration))
}

type ConnectionListener struct {
	clientSessionLock sync.Mutex
	clientSessionMap  map[string]clientSessionRecord
	cleanSessionTick  *time.Ticker
}

func (g *ConnectionListener) addClientSession(ctx SessionContext) {
	g.clientSessionLock.Lock()
	defer g.clientSessionLock.Unlock()
	g.clientSessionMap[ctx.String()] = clientSessionRecord{ctx: ctx, epoch: time.Now()}
}

func (g *ConnectionListener) removeClientSession(ctx SessionContext) {
	g.clientSessionLock.Lock()
	defer g.clientSessionLock.Unlock()
	delete(g.clientSessionMap, ctx.String())
}

func (g *ConnectionListener) StartSession(ctx SessionContext) {
	g.addClientSession(ctx)
}

func (g *ConnectionListener) OnReceiveCommand(ctx SessionContext, message Message) {
	// do nothing
}

func (g *ConnectionListener) OnClose(ctx SessionContext) {
	g.removeClientSession(ctx)
}
