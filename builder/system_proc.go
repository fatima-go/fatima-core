/*
 * Copyright 2023 github.com/fatima-go
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
 * @date 23. 4. 12. 오후 1:41
 */

package builder

import (
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/fatima-go/fatima-core"
)

type FatimaSystemProc struct {
	pid         int
	uid         int
	gid         string
	username    string
	homeDir     string
	programName string
}

func (this *FatimaSystemProc) GetPid() int {
	return this.pid
}

func (this *FatimaSystemProc) GetUid() int {
	return this.uid
}

func (this *FatimaSystemProc) GetProgramName() string {
	return this.programName
}

func (this *FatimaSystemProc) GetUsername() string {
	return this.username
}

func (this *FatimaSystemProc) GetHomeDir() string {
	return this.homeDir
}

func (this *FatimaSystemProc) GetGid() string {
	return this.gid
}

// load process information
// pid, uid, username, homedir, gid, programname
func newSystemProc() fatima.SystemProc {
	proc := new(FatimaSystemProc)
	proc.pid = os.Getpid()
	systemUser, _ := user.Current()
	uid, _ := strconv.ParseInt(systemUser.Uid, 10, 32)
	proc.uid = int(uid)
	proc.username = systemUser.Username
	proc.homeDir = systemUser.HomeDir
	proc.gid = systemUser.Gid

	debugAppName := getDebugAppName()
	if len(debugAppName) > 0 {
		proc.programName = debugAppName
	} else {
		proc.programName = getProgramName()
	}

	return proc
}

var debugappList = [...]string{"-debugapp=", "debugapp="}

func getDebugAppName() string {
	if len(os.Args) == 1 {
		return ""
	}

	for _, v := range os.Args[1:] {
		for _, s := range debugappList {
			if strings.HasPrefix(v, s) {
				return v[len(s):]
			}
		}
	}

	return ""
}

func getProgramName() string {
	var procName string
	args0 := os.Args[0]
	lastIndex := strings.LastIndex(os.Args[0], "/")
	if lastIndex >= 0 {
		procName = args0[lastIndex+1:]
	} else {
		procName = os.Args[0]
	}

	firstIndex := strings.Index(procName, " ")
	if firstIndex > 0 {
		procName = procName[:firstIndex]
	}

	return procName
}
