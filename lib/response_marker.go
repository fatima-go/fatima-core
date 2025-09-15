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
	"bytes"
	"fmt"
	"sync"

	"github.com/fatima-go/fatima-core"
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
	fatimaRuntime.RegisterMeasureUnit(m)
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
