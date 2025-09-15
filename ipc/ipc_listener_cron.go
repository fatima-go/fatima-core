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
 * @date 25. 9. 12. 오전 9:57
 */

package ipc

import (
	"strings"

	log "github.com/fatima-go/fatima-log"
)

func registerCronListener() {
	RegisterIPCSessionListener(newCronListener())
}

func newCronListener() FatimaIPCSessionListener {
	return &CronListener{}
}

type CronListener struct {
}

func (g *CronListener) StartSession(ctx SessionContext) {
	log.Trace("[%s] start session", ctx)
}

func (g *CronListener) OnClose(ctx SessionContext) {
	log.Trace("[%s] on close", ctx)
}

func (g *CronListener) OnReceiveCommand(ctx SessionContext, message Message) {
	log.Trace("IPC command incoming : %s", message)

	if !message.Is(CommandCronExecute) {
		return
	}

	defer ctx.Close()

	if cronRunner == nil {
		// do nothing
		log.Warn("[%s] received cron execute command but cronRunner is not set", ctx)
		return
	}

	log.Warn("IPC process CronExecute : %s", message)
	jobName := AsString(message.Data.GetValue(DataKeyJobName))
	if len(jobName) == 0 {
		log.Warn("[%s] received empty jobName", ctx)
		return
	}
	sample := AsString(message.Data.GetValue(DataKeyJobSample))
	log.Info("[%s] jobName : %s, sample : %s", ctx, jobName, sample)

	var args []string
	if len(sample) > 0 {
		line := strings.TrimSpace(sample)
		if len(line) > 0 {
			args = strings.Split(line, " ")
		}
	}

	log.Trace("[%s] executing : jobName=%s, args=%v", jobName, args)
	go cronRunner(jobName, args)
}
