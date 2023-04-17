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
	"context"
	"fmt"
	"github.com/fatima-go/fatima-core"
	proto "github.com/fatima-go/fatima-core/builder/fatima.message.v1"
	"github.com/fatima-go/fatima-core/monitor"
	"github.com/fatima-go/fatima-log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
	"time"
)

const (
	propPredefineSaturnPort   = "var.saturn.port"
	propPredefineSaturnEnable = "var.saturn.enable"
	valueDefaultAddress       = ":4389"
	maxQueueSize              = 4096
	dropQueueSize             = 1024 // drop if queue fulls at least half size
)

type GrpcSystemNotifyHandler struct {
	fatimaRuntime fatima.FatimaRuntime
	saturnEnabled bool
	saturnAddress string
	conn          *grpc.ClientConn
	queue         chan []byte
}

// NewGrpcSystemNotifyHandler create system notify handler
// fatima process send any event/alarm to saturn via grpc
func NewGrpcSystemNotifyHandler(fatimaRuntime fatima.FatimaRuntime) (monitor.SystemNotifyHandler, error) {
	handler := GrpcSystemNotifyHandler{fatimaRuntime: fatimaRuntime}

	// all (want to deliver) message store to queue
	handler.queue = make(chan []byte, maxQueueSize)
	handler.saturnEnabled = true

	var err error
	handler.saturnAddress, err = buildSaturnServiceAddress(NewPropertyPredefineReader(fatimaRuntime.GetEnv()))
	if err != nil {
		log.Warn("NewGrpcSystemNotifyHandler : %s", err.Error())
		handler.saturnEnabled = false
	}

	go handler.consumeQueue()

	return &handler, nil
}

// consumeQueue send event/alarm message to saturn
func (s *GrpcSystemNotifyHandler) consumeQueue() {
	for notifyItem := range s.queue {
		if len(notifyItem) < 3 {
			continue
		}

		if !s.saturnEnabled {
			continue
		}

		req := proto.SendFatimaMessageRequest{}
		req.JsonString = string(notifyItem)

		for true {
			if s.conn == nil {
				s.connectSaturn()
			}

			if s.conn == nil {
				log.Warn("sleep for connecting to saturn...")
				time.Sleep(time.Second * 5)
				continue
			}

			ok := s.sendToSaturn(req)
			if !ok {
				break
			}

			// success
			break
		}
	}
}

func (s *GrpcSystemNotifyHandler) sendToSaturn(req proto.SendFatimaMessageRequest) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := proto.NewFatimaMessageServiceClient(s.conn).SendFatimaMessage(ctx, &req)
	if err != nil {
		log.Warn("SendFatimaMessage grpc exception : %s", err.Error())

		// maybe grpc relative errors...
		s.conn.Close()
		s.conn = nil
		return false
	}

	if errRes, ok := res.Response.(*proto.SendFatimaMessageResponse_Error); ok {
		log.Warn("SendFatimaMessage error : [%s] %s", errRes.Error.Code, errRes.Error.Desc)
	}

	return true
}

func (s *GrpcSystemNotifyHandler) connectSaturn() {
	log.Warn("connecting to saturn %s", s.saturnAddress)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	gConn, err := grpc.DialContext(
		ctx,
		s.saturnAddress,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Warn("fail to connect saturn : %s", err.Error())
		return
	}

	s.conn = gConn
}

var errSaturnDisabled = fmt.Errorf("saturn disabled from configuration")

func buildSaturnServiceAddress(predefinedReader *PropertyPredefineReader) (string, error) {
	useSaturn, ok := predefinedReader.GetDefine(propPredefineSaturnEnable)
	if ok {
		if strings.TrimSpace(strings.ToLower(useSaturn)) == "false" {
			// will not use saturn...
			return "", errSaturnDisabled
		}
	}
	address, ok := predefinedReader.GetDefine(propPredefineSaturnPort)
	if !ok {
		return valueDefaultAddress, nil
	}

	return address, nil
}

var (
	messageDropFlag = false
)

func (s *GrpcSystemNotifyHandler) enqueueForSending(bytes []byte) {
	if len(s.queue) >= dropQueueSize {
		if !messageDropFlag {
			messageDropFlag = true
			log.Warn("notify handler drop message....")
		}
		return // DROP...
	}

	messageDropFlag = false
	s.queue <- bytes
}

func (s *GrpcSystemNotifyHandler) SendAlarm(level monitor.AlarmLevel, message string) {
	s.enqueueForSending(buildAlarmMessage(s.fatimaRuntime, level, message, ""))
}

func (s *GrpcSystemNotifyHandler) SendAlarmWithCategory(level monitor.AlarmLevel, message string, category string) {
	s.enqueueForSending(buildAlarmMessage(s.fatimaRuntime, level, message, category))
}

func (s *GrpcSystemNotifyHandler) SendEvent(message string, v ...interface{}) {
	s.enqueueForSending(buildEventMessage(s.fatimaRuntime, message, v...))
}

func (s *GrpcSystemNotifyHandler) SendActivity(json interface{}) {
	s.enqueueForSending(buildActivityMessage(s.fatimaRuntime, json))
}
