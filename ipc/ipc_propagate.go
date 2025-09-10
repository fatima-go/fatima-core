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

func propagateSessionStarted(ctx SessionContext) {
	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	for _, v := range ipcSessionListeners {
		go v.StartSession(ctx)
	}
}

func propagateOnReceiveCommand(ctx SessionContext, message Message) {
	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	for _, v := range ipcSessionListeners {
		go v.OnReceiveCommand(ctx, message)
	}
}

func propagateOnClose(ctx SessionContext) {
	ipcSessionListenerLock.Lock()
	defer ipcSessionListenerLock.Unlock()
	for _, v := range ipcSessionListeners {
		go v.OnClose(ctx)
	}
}
