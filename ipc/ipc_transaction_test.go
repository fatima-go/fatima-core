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
 * @date 25. 9. 11. 오전 11:39
 */

package ipc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func beforeTestTransaction() {
	clearAllTransaction()
	setTransactionAliveDuration(time.Second)
	setCleanTransactionTickDuration(time.Second)
}

func TestTransaction(t *testing.T) {
	beforeTestTransaction()

	tid := generateTransactionId()
	assert.True(t, IsAliveTransaction(tid))

	time.Sleep(2 * time.Second)
	assert.False(t, IsAliveTransaction(tid))

	tid = generateTransactionId()
	tid = generateTransactionId()
	assert.Equal(t, 2, countTransaction())

	time.Sleep(2 * time.Second)
	assert.Equal(t, 0, countTransaction())
}
