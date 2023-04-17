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

package lib

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/fatima-go/fatima-log"
	"os"
	"reflect"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	ErrInvalidIndex      = errors.New("mmap: invalid index or ptr")
	ErrInsufficientSpace = errors.New("mmap: insufficient available space")
)

type Mmap struct {
	slice     []byte
	file      *os.File
	data      uintptr
	len       int
	byteOrder binary.ByteOrder
}

func NewMmap(filepath string, length int) (*Mmap, error) {
	m := Mmap{}
	m.byteOrder = getByteOrder()

	var err error
	if _, err = os.Stat(filepath); err != nil {
		if os.IsNotExist(err) {
			m.file, err = os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return nil, err
			}

			err = syscall.Ftruncate(int(m.file.Fd()), int64(length))
			if err != nil {
				return nil, err
			}
			m.len = length
		} else {
			return nil, err
		}
	} else {
		m.file, err = os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
		var fi os.FileInfo
		fi, err = m.file.Stat()
		fileSize := int(fi.Size())
		if length > fileSize {
			return nil, fmt.Errorf("mmap: exist file[%s] size is %d, but required %d", filepath, fileSize, length)
		}
	}

	m.len = length
	m.slice, err = syscall.Mmap(int(m.file.Fd()),
		0,
		m.len,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	header := (*reflect.SliceHeader)(unsafe.Pointer(&m.slice))
	m.data = header.Data

	return &m, nil
}

func (m *Mmap) Write(ptr int, data []byte) error {
	if ptr < 0 {
		return ErrInvalidIndex
	}
	if ptr+len(data) > m.len {
		return ErrInsufficientSpace
	}

	copy(m.slice[ptr:], data)

	//return m.flush(ptr, data)
	return nil
}

func (m *Mmap) WriteUint32(ptr int, v uint32) error {
	if ptr < 0 {
		return ErrInvalidIndex
	}
	if ptr+4 > m.len {
		return ErrInsufficientSpace
	}

	m.byteOrder.PutUint32(m.slice[ptr:], v)
	return nil
}

func (m *Mmap) WriteUint64(ptr int, v uint64) error {
	if ptr < 0 {
		return ErrInvalidIndex
	}
	if ptr+8 > m.len {
		return ErrInsufficientSpace
	}

	m.byteOrder.PutUint64(m.slice[ptr:], v)
	return nil
}

func (m *Mmap) Read(ptr int, dst []byte) error {
	if ptr < 0 {
		return ErrInvalidIndex
	}

	if ptr+len(dst) > m.len {
		return ErrInsufficientSpace
	}

	copy(dst, m.slice[ptr:])
	return nil
}

func (m *Mmap) ReadUint32(ptr int) (uint32, error) {
	if ptr < 0 {
		return 0, ErrInvalidIndex
	}
	if ptr+4 > m.len {
		return 0, ErrInsufficientSpace
	}

	return m.byteOrder.Uint32(m.slice[ptr:]), nil
}

func (m *Mmap) ReadUint64(ptr int) (uint64, error) {
	if ptr < 0 {
		return 0, ErrInvalidIndex
	}
	if ptr+8 > m.len {
		return 0, ErrInsufficientSpace
	}

	return m.byteOrder.Uint64(m.slice[ptr:]), nil
}

func (m *Mmap) Flush() error {
	return m.FlushRegion(0, m.len)
}

func (m *Mmap) FlushRegion(ptr int, length int) error {
	if ptr < 0 {
		return ErrInvalidIndex
	}

	addr := m.data + uintptr(ptr)
	len := uintptr(length)
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, addr, len, syscall.MS_SYNC)
	if errno != 0 {
		return syscall.Errno(errno)
	}
	return nil
}

func (m *Mmap) Close() error {
	if m.file != nil {
		log.Trace("mmap file [%s] closed", m.file.Name())
		m.file.Close()
		m.file = nil
		m.unmap()
	}
	return nil
}

func (m *Mmap) unmap() error {
	_, _, errno := syscall.Syscall(syscall.SYS_MUNMAP, m.data, uintptr(m.len), 0)
	if errno != 0 {
		return syscall.Errno(errno)
	}
	return nil
}

func (m *Mmap) GetByteOrder() binary.ByteOrder {
	return m.byteOrder
}

func getByteOrder() binary.ByteOrder {
	switch runtime.GOARCH {
	case "AMD64", "X86", "ARM":
		return binary.LittleEndian
	default:
		return binary.BigEndian
	}
}
