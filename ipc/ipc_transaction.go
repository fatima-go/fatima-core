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
 * @date 25. 9. 11. 오전 11:26
 */

package ipc

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatima-go/fatima-core/lib"
	log "github.com/fatima-go/fatima-log"
)

var transactionCounter int64

func generateTransactionId() string {
	return addTransaction(buildTransactionId())
}

func buildTransactionId() string {
	counter := atomic.AddInt64(&transactionCounter, 1)
	return fmt.Sprintf("%s%02d",
		lib.RandomAlphanumeric(8),
		counter%100)
}

var cleanTransactionOnce sync.Once

func addTransaction(id string) string {
	cleanTransactionOnce.Do(startCleanTransactionTick)
	transactionLock.Lock()
	defer transactionLock.Unlock()
	transactionMap[id] = time.Now().Add(getTransactionAliveDuration())
	return id
}

func IsAliveTransaction(id string) bool {
	transactionLock.Lock()
	defer transactionLock.Unlock()
	until, found := transactionMap[id]
	if !found {
		return false
	}
	if _, ok := transactionMap[id]; !ok {
		return false
	}
	if time.Now().Before(until) {
		return true
	}
	delete(transactionMap, id)
	return false
}

// for testing purpose
func countTransaction() int {
	transactionLock.Lock()
	defer transactionLock.Unlock()
	return len(transactionMap)
}

// for testing purpose
func clearAllTransaction() {
	transactionLock.Lock()
	defer transactionLock.Unlock()
	transactionMap = make(map[string]time.Time)
}

var transactionAliveDuration = time.Minute
var cleanTransactionTickDuration = time.Second

// for testing purpose
func setTransactionAliveDuration(duration time.Duration) {
	transactionAliveDuration = duration
}

// for testing purpose
func setCleanTransactionTickDuration(duration time.Duration) {
	cleanTransactionTickDuration = duration
	restartCleanTransactionTick()
}

func getTransactionAliveDuration() time.Duration {
	if transactionAliveDuration < time.Second {
		return time.Second
	}
	return transactionAliveDuration
}

func getCleanTransactionTickDuration() time.Duration {
	if cleanTransactionTickDuration < time.Second {
		return time.Second
	}
	return cleanTransactionTickDuration
}

var transactionMap = make(map[string]time.Time)

var transactionLock sync.Mutex
var cleanTransactionTick *time.Ticker

func startCleanTransactionTick() {
	if cleanTransactionTick != nil {
		cleanTransactionTick.Stop()
	}
	cleanTransactionTick = time.NewTicker(getCleanTransactionTickDuration())
	if log.IsTraceEnabled() {
		log.Trace("[IPC] clean transaction tick started. duration=%d secs", int(cleanTransactionTickDuration.Seconds()))
	}

	go func() {
		for range cleanTransactionTick.C {
			transactionLock.Lock()
			for id, until := range transactionMap {
				if time.Now().After(until) {
					delete(transactionMap, id)
					log.Trace("[IPC] transaction %s removed", id)
				}
			}
			transactionLock.Unlock()
		}
	}()
}

// for testing purpose
func restartCleanTransactionTick() {
	if cleanTransactionTick == nil {
		return
	}
	startCleanTransactionTick()
}
