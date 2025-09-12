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
 * @date 25. 9. 9. 오전 10:54
 *
 */

package ipc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	log "github.com/fatima-go/fatima-log"
)

/*
{
  "initiator" : {
    {
      "process" : "mypgm",
      "command" : "TRANSACTION_VERIFY",
      "sock" : "/tmp/fatima.mypgm.312.sock"
    },
    "data" : {
      {
        "transaction" : "1234567890"
      }
    }
  }
}
*/

const (
	CommandGoaway                = "GOAWAY"
	CommandTransactionVerify     = "TRANSACTION_VERIFY"
	CommandTransactionVerifyDone = "TRANSACTION_VERIFY_DONE"
	CommandGoawayStart           = "GOAWAY_START"
	CommandGoawayDone            = "GOAWAY_DONE"
	CommandCronExecute           = "CRON_EXECUTE"
	DataKeyTransaction           = "transaction"
	DataKeyVerify                = "verify"
	DataKeyJobName               = "job"
	DataKeyJobSample             = "sample"
)

func newMessage(command string) Message {
	m := Message{}
	m.Initiator.Command = command
	m.Initiator.Process = envProvideHelper.getProgramName()
	m.Initiator.Sock = envProvideHelper.buildAddress()
	return m
}

func NewMessageGoaway() Message {
	m := newMessage(CommandGoaway)
	m.Data = JsonBody{DataKeyTransaction: generateTransactionId()}
	return m
}

func NewMessageGoawayStart(transactionId string) Message {
	m := newMessage(CommandGoawayStart)
	m.Data = JsonBody{DataKeyTransaction: generateTransactionId()}
	return m
}

func NewMessageGoawayDone(transactionId string) Message {
	m := newMessage(CommandGoawayDone)
	m.Data = JsonBody{DataKeyTransaction: generateTransactionId()}
	return m
}

func NewMessageTransactionVerify(transactionId string) Message {
	m := newMessage(CommandTransactionVerify)
	m.Data = JsonBody{DataKeyTransaction: transactionId}
	return m
}

func NewMessageTransactionVerifyDone(transactionId string, result bool) Message {
	m := newMessage(CommandTransactionVerifyDone)
	m.Data = JsonBody{DataKeyTransaction: transactionId, DataKeyVerify: result}
	return m
}

func NewMessageCronExecute(jobName, sample string) Message {
	m := newMessage(CommandCronExecute)
	m.Data = JsonBody{DataKeyJobName: jobName, DataKeyJobSample: sample}
	return m
}

type Message struct {
	Initiator Initiator `json:"initiator"`
	Data      JsonBody  `json:"data,omitempty"`
}

func (m Message) String() string {
	return fmt.Sprintf("initiator=[%s], data=%v", m.Initiator, m.Data)
}

func (m Message) Is(command string) bool {
	return m.Initiator.Command == command
}

type Initiator struct {
	Process string `json:"process"`
	Command string `json:"command"`
	Sock    string `json:"sock"`
}

func (i Initiator) String() string {
	return fmt.Sprintf("P:%s|C:%s|S:%s", i.Process, i.Command, i.Sock)
}

func parseMessage(d []byte) (Message, error) {
	message := Message{}
	err := json.Unmarshal(d, &message)
	if err != nil {
		return message, fmt.Errorf("fail to parse initiator : %s", err.Error())
	}

	return message, nil
}

type JsonBody map[string]interface{}

func (j JsonBody) GetValue(findingKey string) interface{} {
	keyPath := strings.Split(findingKey, ".")
	var currentDepth map[string]interface{}
	currentDepth = j
	var currentList []interface{}
	for i, key := range keyPath {
		if isNumeric(key) && currentList != nil {
			index, _ := strconv.Atoi(key)
			if index >= len(currentList) {
				log.Debug("[%s] JsonBody.GetValue() :  list out of index (expect:%d, total len:%d)",
					findingKey, index, len(currentList))
				return nil
			}
			if shouldBeMap, isMap := currentList[index].(map[string]interface{}); isMap {
				currentDepth = shouldBeMap
				currentList = nil
				continue
			} else {
				log.Debug("[%s] JsonBody.GetValue() :  index %d is not map type",
					findingKey, index)
				return nil
			}
		}

		found, ok := currentDepth[key]
		if !ok {
			return nil // not found
		}

		if i == len(keyPath)-1 {
			// 마지막 element라면... 다 찾은것이므로..
			return found
		}

		switch v := found.(type) {
		case map[string]interface{}:
			currentDepth = v // map의 경우 다음 키를 찾기 위해 진행
		case []interface{}:
			currentList = v // array일 경우 다음 키패스가 "숫자"형태여야 한다
		default:
			log.Debug("[%s] JsonBody.GetValue() :  key is reached to unsupported type (%s)", findingKey, key)
			return nil
		}
	}

	return nil
}

func isNumeric(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func AsString(src interface{}) string {
	if src == nil {
		return ""
	}

	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	rv := reflect.ValueOf(src)
	switch rv.Kind() {
	case reflect.Float64:
		return fmt.Sprintf("%d", int(rv.Float()))
		//return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	default:
		return fmt.Sprintf("%v", src)
	}
}

func AsBool(src interface{}) bool {
	if src == nil {
		return false
	}

	switch v := src.(type) {
	case string:
		return toBool(v)
	case []byte:

		return toBool(string(v))
	}
	rv := reflect.ValueOf(src)
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	default:
		return false
	}
}

func toBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes" || s == "y" || s == "t"
}
