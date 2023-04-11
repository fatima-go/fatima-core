//
// Copyright (c) 2018 SK Planet.
// All right reserved.
//
// This software is the confidential and proprietary information of K Planet.
// You shall not disclose such Confidential Information and
// shall use it only in accordance with the terms of the license agreement
// you entered into with SK Planet.
//
//
// @project fatima
// @author 1100282
// @date 2018. 9. 12. PM 7:30
//

package lib

import (
	"bytes"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"sync"
)

type ResponseMarker interface {
	Mark(score int)
}

const (
	defaultMax  = 10000
	defaultPart = 10
)

func NewResponseMarker(fatimaRuntime fatima.FatimaRuntime, name string) ResponseMarker {
	return NewCustomResponseMarker(fatimaRuntime, name, defaultMax, defaultPart)
}

func NewCustomResponseMarker(fatimaRuntime fatima.FatimaRuntime, name string, max, part int) ResponseMarker {
	m := &basicResponseMarker{}
	m.name = name
	if max < 1000 {
		max = 1000
	}
	m.max = max

	if part < 1 {
		part = 1
	}
	m.part = part
	m.build()
	fatimaRuntime.RegistMeasureUnit(m)
	return m
}

type basicResponseMarker struct {
	mutex  sync.Mutex
	name   string
	max    int
	part   int
	bunch  int
	list   []int
	labels []string
}

func (b *basicResponseMarker) build() {
	size := b.part + 1
	b.list = make([]int, size)
	b.labels = make([]string, size)
	b.bunch = b.max / b.part
	for i := 0; i < b.part; i++ {
		b.labels[i] = fmt.Sprintf("%5d", (i*b.bunch)+b.bunch)
	}
	b.labels[b.part] = "TOTAL"
}

func (b *basicResponseMarker) reset() {
	for i := 0; i < len(b.labels); i++ {
		b.list[i] = 0
	}
}

func (b *basicResponseMarker) Mark(score int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	defer func() {
		b.list[b.part]++
	}()

	if score <= 0 {
		b.list[0]++
		return
	}

	if score >= b.max {
		b.list[len(b.list)-1]++
		return
	}

	pos := score / b.bunch
	b.list[pos]++
}

func (b *basicResponseMarker) GetKeyName() string {
	return b.name
}

func (b *basicResponseMarker) GetMeasure() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	defer b.reset()

	buff := bytes.Buffer{}
	for i := 0; i < len(b.labels); i++ {
		buff.WriteString(b.labels[i])
		buff.WriteString("|")
	}
	buff.WriteString("\n")
	for i := 0; i < len(b.list); i++ {
		buff.WriteString(fmt.Sprintf("%5d", b.list[i]))
		buff.WriteString("|")
	}

	return buff.String()
}
