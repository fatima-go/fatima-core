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

package fatima

/**
 * @author jin.freestyle@gmail.com
 *
 */

type Process interface {
	// Pid is the process ID for this process.
	Pid() int

	// PPid is the parent process ID for this process.
	PPid() int

	// Executable name running this process. This is not a path to the
	// executable.
	Executable() string
}

type PlatformSupport interface {
	// EnsureSingleInstance ensure (only 1) single process running
	EnsureSingleInstance(proc SystemProc) error
	// check process is running or not
	CheckProcessRunningByPid(procName string, pid int) bool
	// GetProcesses load all process list
	GetProcesses() ([]Process, error)
	// Dup3 duplicate fd
	Dup3(oldfd int, newfd int, flags int) (err error)
}
