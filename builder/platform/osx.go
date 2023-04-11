//go:build darwin
// +build darwin

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with p work for additional information
 * regarding copyright ownership.  The ASF licenses p file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use p file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 * @project fatima
 * @author DeockJin Chung (jin.freestyle@gmail.com)
 * @date 22. 8. 31. 오전 11:28
 */

package platform

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

/**
 * @author jin.freestyle@gmail.com
 * https://github.com/mitchellh/go-ps
 */

type OSPlatform struct {
}

func (t *OSPlatform) EnsureSingleInstance(proc fatima.SystemProc) error {
	ps, err := t.GetProcesses()
	if err != nil {
		return nil
	}

	for _, p := range ps {
		if p.Pid() == proc.GetPid() {
			continue
		}

		bin := strings.Split(p.Executable(), " ")
		if bin[0] == proc.GetProgramName() {
			return errors.New("aleady process running...")
		}
	}

	return nil
}

func (t *OSPlatform) GetProcesses() ([]fatima.Process, error) {
	buf, err := darwinSyscall()
	if err != nil {
		return nil, err
	}

	procs := make([]*kinfoProc, 0, 50)
	k := 0
	for i := _KINFO_STRUCT_SIZE; i < buf.Len(); i += _KINFO_STRUCT_SIZE {
		proc := &kinfoProc{}
		err = binary.Read(bytes.NewBuffer(buf.Bytes()[k:i]), binary.LittleEndian, proc)
		if err != nil {
			return nil, err
		}

		k = i
		procs = append(procs, proc)
	}

	darwinProcs := make([]fatima.Process, len(procs))
	for i, p := range procs {
		darwinProcs[i] = &DarwinProcess{
			pid:    int(p.Pid),
			ppid:   int(p.PPid),
			binary: darwinCstring(p.Comm),
		}
	}

	return darwinProcs, nil
}

// Dup3 is not available on Darwin, so we use Dup2 instead.
func (t *OSPlatform) Dup3(oldfd int, newfd int, flags int) (err error) {
	if flags != 0 {
		return fmt.Errorf("darwin does not support dup3 flags")
	}
	return syscall.Dup2(oldfd, newfd)
}

type DarwinProcess struct {
	pid    int
	ppid   int
	binary string
}

func (p *DarwinProcess) Pid() int {
	return p.pid
}

func (p *DarwinProcess) PPid() int {
	return p.ppid
}

func (p *DarwinProcess) Executable() string {
	return p.binary
}

func darwinCstring(s [16]byte) string {
	i := 0
	for _, b := range s {
		if b != 0 {
			i++
		} else {
			break
		}
	}

	return string(s[:i])
}

func darwinSyscall() (*bytes.Buffer, error) {
	user, _ := user.Current()
	uid, _ := strconv.ParseInt(user.Uid, 10, 32)
	mib := [4]int32{_CTRL_KERN, _KERN_PROC, _KERN_PROC_UID, int32(uid)}
	size := uintptr(0)

	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	bs := make([]byte, size)
	_, _, errno = syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		uintptr(unsafe.Pointer(&bs[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	return bytes.NewBuffer(bs[0:size]), nil
}

const (
	_CTRL_KERN         = 1
	_KERN_PROC         = 14
	_KERN_PROC_ALL     = 0 // everything
	_KERN_PROC_PID     = 1 // by process id
	_KERN_PROC_PGRP    = 2 // by process group id
	_KERN_PROC_SESSION = 3 // by session of pid
	_KERN_PROC_TTY     = 4 // by controlling tty
	_KERN_PROC_UID     = 5 // by effective uid
	_KERN_PROC_RUID    = 6 // by real uid
	_KERN_PROC_LCID    = 7 // by login context id
	_KINFO_STRUCT_SIZE = 648
)

type kinfoProc struct {
	_    [40]byte
	Pid  int32
	_    [199]byte
	Comm [16]byte
	_    [301]byte
	PPid int32
	_    [84]byte
}
