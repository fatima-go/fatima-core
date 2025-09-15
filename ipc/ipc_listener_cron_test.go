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
 * @date 25. 9. 12. 오후 3:52
 */

package ipc

import (
	"fmt"
	"testing"
	"time"

	log "github.com/fatima-go/fatima-log"
	"github.com/stretchr/testify/assert"
)

var cronSimulator = &dummyCronRunner{}

func beforeTestProviderForCronListener() {
	envProvideHelper.getPid = mockGetPid
	envProvideHelper.getSockDir = mockGetSockDir
	envProvideHelper.getProgramName = mockGetJunoProgramName
	envProvideHelper.buildAddress = mockBuildAddress
	cronRunner = cronSimulator.Rerun
	cronSimulator.reset()
}

// runJunoServer juno 시뮬레이터를 IPC 서버로 등록
func runJunoServerForCronTest() {
	beforeTestProviderForCronListener()
	RegisterIPCSessionListener(newCronListener())
	startIPCServer()
}

// TestCronExecuteOnyJobName jobName만 전달하는 정상 케이스
func TestCronExecuteOnyJobName(t *testing.T) {
	runJunoServerForCronTest()
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)

	err := sendCronTestMessage("my.batch", "")
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	time.Sleep(time.Millisecond * 100)

	assert.True(t, cronSimulator.called)
	assert.Equal(t, "my.batch", cronSimulator.jobName)
	assert.Equal(t, 0, len(cronSimulator.args))
}

// TestCronExecuteWithInvalidJobName jobName이 없는 예외 케이스
func TestCronExecuteWithInvalidJobName(t *testing.T) {
	runJunoServerForCronTest()
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)

	err := sendCronTestMessage("", "")
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	time.Sleep(time.Millisecond * 100)

	assert.False(t, cronSimulator.called)
	assert.Equal(t, "", cronSimulator.jobName)
	assert.Equal(t, 0, len(cronSimulator.args))
}

// TestCronExecuteWithArgs jobName과 파라미터 전달 정상 케이스
func TestCronExecuteWithArgs(t *testing.T) {
	runJunoServerForCronTest()
	defer stopIPCServer()

	time.Sleep(time.Millisecond * 100)

	err := sendCronTestMessage("my.batch", "hello world")
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	time.Sleep(time.Millisecond * 100)

	assert.True(t, cronSimulator.called)
	assert.Equal(t, "my.batch", cronSimulator.jobName)
	assert.Equal(t, 2, len(cronSimulator.args))
	assert.Equal(t, "hello", cronSimulator.args[0])
	assert.Equal(t, "world", cronSimulator.args[1])
}

func sendCronTestMessage(jobName, sample string) error {
	junoClient, err := NewFatimaIPCClientSession(junoProgramName)
	if err != nil {
		return fmt.Errorf("cannot make connection to %s : %s", junoProgramName, err.Error())
	}

	defer junoClient.Disconnect()

	err = junoClient.SendCommand(NewMessageCronExecute(jobName, sample))
	if err != nil {
		return fmt.Errorf("fail to send cron execute : %s", err.Error())
	}

	return nil
}

type dummyCronRunner struct {
	called  bool
	jobName string
	args    []string
}

func (d *dummyCronRunner) reset() {
	d.called = false
	d.jobName = ""
	d.args = nil
}

func (d *dummyCronRunner) Rerun(jobName string, args []string) {
	log.Trace("called rerun name=%s, args=%v", jobName, args)
	d.called = true
	d.jobName = jobName
	d.args = args
}
