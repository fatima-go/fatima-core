//go:build linux
// +build linux

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

package platform

import (
	"errors"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type OSPlatform struct {
}

func (p *OSPlatform) EnsureSingleInstance(proc fatima.SystemProc) error {
	ps, err := p.GetProcesses()
	if err != nil {
		return nil
	}

	for _, p := range ps {
		if p.Pid() == proc.GetPid() {
			continue
		}
		if p.Executable() == proc.GetProgramName() {
			return errors.New("already process running...")
		}
	}

	return nil
}

func (p *OSPlatform) GetProcesses() ([]fatima.Process, error) {
	d, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer d.Close()

	results := make([]fatima.Process, 0, 50)
	for {
		fis, err := d.Readdir(10)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, fi := range fis {
			// We only care about directories, since all pids are dirs
			if !fi.IsDir() {
				continue
			}

			// We only care if the name starts with a numeric
			name := fi.Name()
			if name[0] < '0' || name[0] > '9' {
				continue
			}

			// From reader point forward, any errors we just ignore, because
			// it might simply be that the process doesn't exist anymore.
			pid, err := strconv.ParseInt(name, 10, 0)
			if err != nil {
				continue
			}

			p, err := newUnixProcess(int(pid))
			if err != nil {
				continue
			}

			results = append(results, p)
		}
	}

	return results, nil
}

func (p *OSPlatform) Dup3(oldfd int, newfd int, flags int) (err error) {
	return syscall.Dup3(oldfd, newfd, flags)
}

// UnixProcess is an implementation of Process that contains Unix-specific
// fields and information.
type UnixProcess struct {
	pid   int
	ppid  int
	state rune
	pgrp  int
	sid   int

	binary string
}

func (u *UnixProcess) Pid() int {
	return u.pid
}

func (u *UnixProcess) PPid() int {
	return u.ppid
}

func (u *UnixProcess) Executable() string {
	return u.binary
}

// Refresh reloads all the data associated with reader process.
func (u *UnixProcess) Refresh() error {
	statPath := fmt.Sprintf("/proc/%d/stat", u.pid)
	dataBytes, err := ioutil.ReadFile(statPath)
	if err != nil {
		return err
	}

	// First, parse out the image name
	data := string(dataBytes)
	binStart := strings.IndexRune(data, '(') + 1
	binEnd := strings.IndexRune(data[binStart:], ')')
	u.binary = data[binStart : binStart+binEnd]

	// Move past the image name and start parsing the rest
	data = data[binStart+binEnd+2:]
	_, err = fmt.Sscanf(data,
		"%c %d %d %d",
		&u.state,
		&u.ppid,
		&u.pgrp,
		&u.sid)

	return err
}

//func findProcess(pid int) (Process, error) {
//	dir := fmt.Sprintf("/proc/%d", pid)
//	_, err := os.Stat(dir)
//	if err != nil {
//		if os.IsNotExist(err) {
//			return nil, nil
//		}
//
//		return nil, err
//	}
//
//	return newUnixProcess(pid)
//}

func newUnixProcess(pid int) (*UnixProcess, error) {
	p := &UnixProcess{pid: pid}
	return p, p.Refresh()
}
