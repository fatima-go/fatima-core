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
 * @date 25. 9. 9. 오전 10:47
 *
 */

package ipc

import "sync"

var ipcSessionListeners = make([]chan SessionEvent, 0)
var ipcSessionListenerLock sync.Mutex

func RegisterIPCSessionListener(listener FatimaIPCSessionListener) {
	sessionEventChan := make(chan SessionEvent, 8)

	ipcSessionListenerLock.Lock()
	ipcSessionListeners = append(ipcSessionListeners, sessionEventChan)
	ipcSessionListenerLock.Unlock()

	go func() {
		for event := range sessionEventChan {
			switch event.eventType {
			case SessionEventStart:
				listener.StartSession(event.ctx)
			case SessionEventReceiveCommand:
				listener.OnReceiveCommand(event.ctx, event.message)
			case SessionEventClose:
				listener.OnClose(event.ctx)
			}
		}
	}()
}

func closeAllSessionListeners() {
	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	for _, listenerChan := range ipcSessionListeners {
		close(listenerChan)
	}
	ipcSessionListeners = nil
}

type SessionEventType uint8

const (
	SessionEventStart SessionEventType = iota
	SessionEventReceiveCommand
	SessionEventClose
)

type SessionEvent struct {
	eventType SessionEventType
	ctx       SessionContext
	message   Message
}

func propagateSessionStarted(ctx SessionContext) {
	sessionEvent := SessionEvent{eventType: SessionEventStart, ctx: ctx}

	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	for _, listenerChan := range ipcSessionListeners {
		listenerChan <- sessionEvent
	}
}

func propagateOnReceiveCommand(ctx SessionContext, message Message) {
	sessionEvent := SessionEvent{eventType: SessionEventReceiveCommand, ctx: ctx, message: message}

	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	for _, listenerChan := range ipcSessionListeners {
		listenerChan <- sessionEvent
	}
}

func propagateOnClose(ctx SessionContext) {
	sessionEvent := SessionEvent{eventType: SessionEventClose, ctx: ctx}

	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	for _, listenerChan := range ipcSessionListeners {
		listenerChan <- sessionEvent
	}
}
