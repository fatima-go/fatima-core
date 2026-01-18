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

package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/monitor"
)

type MockFatimaRuntime struct {
	env           fatima.FatimaEnv
	config        fatima.Config
	packaging     fatima.Packaging
	systemStatus  monitor.FatimaSystemStatus
	notifyHandler monitor.SystemNotifyHandler
}

func NewMockFatimaRuntime() *MockFatimaRuntime {
	return &MockFatimaRuntime{
		env:           NewMockFatimaEnv(""),
		config:        NewMockConfig(),
		packaging:     NewMockPackaging(),
		systemStatus:  NewMockSystemStatus(),
		notifyHandler: NewMockSystemNotifyHandler(),
	}
}

func (m *MockFatimaRuntime) SetEnv(env fatima.FatimaEnv) *MockFatimaRuntime {
	m.env = env
	return m
}

func (m *MockFatimaRuntime) SetConfig(config fatima.Config) *MockFatimaRuntime {
	m.config = config
	return m
}

func (m *MockFatimaRuntime) SetPackaging(packaging fatima.Packaging) *MockFatimaRuntime {
	m.packaging = packaging
	return m
}

func (m *MockFatimaRuntime) SetSystemStatus(status monitor.FatimaSystemStatus) *MockFatimaRuntime {
	m.systemStatus = status
	return m
}

func (m *MockFatimaRuntime) SetSystemNotifyHandler(handler monitor.SystemNotifyHandler) *MockFatimaRuntime {
	m.notifyHandler = handler
	return m
}

func (m *MockFatimaRuntime) GetEnv() fatima.FatimaEnv {
	return m.env
}

func (m *MockFatimaRuntime) GetConfig() fatima.Config {
	return m.config
}

func (m *MockFatimaRuntime) GetPackaging() fatima.Packaging {
	return m.packaging
}

func (m *MockFatimaRuntime) GetSystemStatus() monitor.FatimaSystemStatus {
	return m.systemStatus
}

func (m *MockFatimaRuntime) GetSystemNotifyHandler() monitor.SystemNotifyHandler {
	return m.notifyHandler
}

func (m *MockFatimaRuntime) IsRunning() bool {
	return true
}

func (m *MockFatimaRuntime) Register(component fatima.FatimaComponent) {
}

func (m *MockFatimaRuntime) RegisterSystemHAAware(aware monitor.FatimaSystemHAAware) {
}

func (m *MockFatimaRuntime) RegisterSystemPSAware(aware monitor.FatimaSystemPSAware) {
}

func (m *MockFatimaRuntime) RegisterMeasureUnit(unit monitor.SystemMeasurable) {
}

func (m *MockFatimaRuntime) Run() {
}

func (m *MockFatimaRuntime) Stop() {
}

type MockConfig struct {
	m map[string]string
}

func NewMockConfigWithMap(m map[string]string) *MockConfig {
	return &MockConfig{m: m}
}

func NewMockConfig() *MockConfig {
	m := make(map[string]string)

	// find project root
	wd, _ := os.Getwd()
	root, found := findProjectRoot(wd) // defined in mock_fatima_env.go
	if !found {
		return &MockConfig{m: m}
	}

	resourcesDir := filepath.Join(root, "resources")

	// determine profile
	profile := os.Getenv(fatima.ENV_FATIMA_PROFILE)
	if profile == "" {
		profile = "local"
	}

	// check application.{profile}.properties
	propFile := filepath.Join(resourcesDir, fmt.Sprintf("application.%s.properties", profile))
	if _, err := os.Stat(propFile); os.IsNotExist(err) {
		// check application.properties
		propFile = filepath.Join(resourcesDir, "application.properties")
	}

	if _, err := os.Stat(propFile); err == nil {
		props, err := readMockProperties(propFile)
		if err == nil {
			m = props
		}
	}

	return &MockConfig{m: m}
}

func readMockProperties(path string) (map[string]string, error) {
	resolved := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var line string
	var idx int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line = strings.Trim(scanner.Text(), " ")
		if strings.HasPrefix(line, "#") || len(line) < 3 {
			continue
		}
		idx = strings.Index(line, "#")
		if idx > 0 {
			if line[idx-1] == ' ' {
				line = line[:idx]
			}
		}
		idx = strings.Index(line, "=")
		if idx < 1 {
			continue
		}
		resolved[line[:idx]] = line[idx+1:]
	}

	return resolved, nil
}

func (c *MockConfig) GetValue(key string) (string, bool) {
	v, ok := c.m[key]
	return v, ok
}

func (c *MockConfig) GetString(key string) (string, error) {
	v, ok := c.m[key]
	if !ok {
		return "", fmt.Errorf("not found key : %s", key)
	}
	return v, nil
}

func (c *MockConfig) GetInt(key string) (int, error) {
	return 0, nil
}

func (c *MockConfig) GetBool(key string) (bool, error) {
	return false, nil
}

// MockPackaging
type MockPackaging struct {
	Name  string
	Host  string
	Group string
}

func NewMockPackaging() *MockPackaging {
	return &MockPackaging{
		Name:  "default",
		Host:  "localhost",
		Group: "dev",
	}
}

func (p *MockPackaging) GetName() string {
	return p.Name
}

func (p *MockPackaging) GetHost() string {
	return p.Host
}

func (p *MockPackaging) GetGroup() string {
	return p.Group
}

// MockSystemStatus
type MockSystemStatus struct {
	psStatus monitor.PSStatus
	haStatus monitor.HAStatus
}

func NewMockSystemStatus() *MockSystemStatus {
	return &MockSystemStatus{
		psStatus: monitor.PS_STATUS_PRIMARY,
		haStatus: monitor.HA_STATUS_ACTIVE,
	}
}

func (s *MockSystemStatus) GetPSStatus() monitor.PSStatus {
	return s.psStatus
}

func (s *MockSystemStatus) GetHAStatus() monitor.HAStatus {
	return s.haStatus
}

func (s *MockSystemStatus) IsActive() bool {
	return s.haStatus == monitor.HA_STATUS_ACTIVE
}

func (s *MockSystemStatus) IsPrimary() bool {
	return s.psStatus == monitor.PS_STATUS_PRIMARY
}

// MockSystemNotifyHandler
type MockSystemNotifyHandler struct {
}

func NewMockSystemNotifyHandler() *MockSystemNotifyHandler {
	return &MockSystemNotifyHandler{}
}

func (h *MockSystemNotifyHandler) SendAlarm(level monitor.AlarmLevel, action monitor.ActionType, message string) {
	fmt.Printf("[MockNotify] ALARM level=%s action=%s message=%s\n", level, action, message)
}

func (h *MockSystemNotifyHandler) SendAlarmWithCategory(level monitor.AlarmLevel, action monitor.ActionType, message string, category string) {
	fmt.Printf("[MockNotify] ALARM level=%s action=%s category=%s message=%s\n", level, action, category, message)
}

func (h *MockSystemNotifyHandler) SendActivity(jsonActivity interface{}) {
	b, _ := json.Marshal(jsonActivity)
	fmt.Printf("[MockNotify] ACTIVITY %s\n", string(b))
}

func (h *MockSystemNotifyHandler) SendEvent(message string, v ...interface{}) {
	msg := fmt.Sprintf(message, v...)
	fmt.Printf("[MockNotify] EVENT %s\n", msg)
}

// SetFatimaRuntimeForTest sets the fatima runtime for testing purposes.
func SetFatimaRuntimeForTest(runtime fatima.FatimaRuntime) {
	process = runtime
}
