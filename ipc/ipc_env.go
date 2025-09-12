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
 * @date 25. 9. 9. 오후 3:35
 *
 */

package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type provideFunc func() string
type providePidFunc func(string) (int, error)

type envProvider struct {
	getPid         providePidFunc
	getSockDir     provideFunc
	buildAddress   provideFunc
	getProgramName provideFunc
}

var envProvideHelper = envProvider{}

func init() {
	envProvideHelper.getPid = getPid
	envProvideHelper.getSockDir = getSockDir
	envProvideHelper.buildAddress = buildAddress
	envProvideHelper.getProgramName = getProgramName
}

func getProgramName() string {
	return fatimaRuntime.GetEnv().GetSystemProc().GetProgramName()
}

func getSockDir() string {
	return fatimaRuntime.GetEnv().GetFolderGuide().GetAppProcFolder()
}

func buildAddress() string {
	return buildAddressForProcess(
		envProvideHelper.getSockDir(),
		envProvideHelper.getProgramName(),
		fatimaRuntime.GetEnv().GetSystemProc().GetPid(),
	)
}

func buildAddressForProcess(sockDir, procName string, pid int) string {
	return filepath.Join(sockDir,
		fmt.Sprintf("%s%s.%d.sock",
			sockFilePrefix,
			procName,
			pid),
	)
}

func getPid(proc string) (int, error) {
	pidFile := fmt.Sprintf("%s/app/%s/proc/%s.pid",
		fatimaRuntime.GetEnv().GetFolderGuide().GetFatimaHome(),
		proc,
		proc,
	)
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return 0, fmt.Errorf("not found pid file : %s", pidFile)
	}

	b, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("fail to read pid file : %s\n", err.Error())
	}

	content := strings.Trim(string(b), "\r\n\t ")
	pid, err := strconv.Atoi(content)
	if err != nil {
		return 0, fmt.Errorf("invalid pid content : [%s]\n", string(b))
	}
	return pid, nil
}
