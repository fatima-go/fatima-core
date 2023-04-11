//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with p work for additional information
// regarding copyright ownership.  The ASF licenses p file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use p file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// @project fatima
// @author DeockJin Chung (jin.freestyle@gmail.com)
// @date 2017. 3. 6. PM 7:42
//

package builder

import (
	"fmt"
	"github.com/fatima-go/fatima-core"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type buildtinVariable string

type variableValue struct {
	key   buildtinVariable
	value string
}

func (this variableValue) getValue() string {
	switch this.key {
	case BUILTIN_VARIABLE_YYYYMM:
		return time.Now().Format("200601")
	case BUILTIN_VARIABLE_YYYYMMDD:
		return time.Now().Format("20060102")
	}
	return this.value
}

type PropertyPredefineReader struct {
	env              fatima.FatimaEnv
	builtinVariables []variableValue

	// defines service key/value properties
	defines  map[string]string
	replacer *strings.Replacer
}

// NewPropertyPredefineReader serving fatima package global(shared) properties
func NewPropertyPredefineReader(env fatima.FatimaEnv) *PropertyPredefineReader {
	instance := new(PropertyPredefineReader)
	instance.env = env
	instance.defines = make(map[string]string)

	// create builtin properties
	instance.buildBuiltin()

	// serving fatima global configuration properties as variable
	instance.prepareMatchers()
	return instance
}

func (reader *PropertyPredefineReader) ResolvePredefine(value string) string {
	return reader.replacer.Replace(value)
}

func (reader *PropertyPredefineReader) GetDefine(key string) (string, bool) {
	v, ok := reader.defines[fmt.Sprintf("${%s}", key)]
	return v, ok
}

// buildBuiltin create builtin properties
// e.g) ${var.builtin.fatima.home}, ${var.builtin.local.ipaddress}
func (reader *PropertyPredefineReader) buildBuiltin() {
	reader.builtinVariables = make([]variableValue, 0)

	reader.appendBuiltinVar(variableValue{BUILTIN_VARIABLE_HOME, reader.env.GetSystemProc().GetHomeDir()})
	reader.appendBuiltinVar(variableValue{BUILTIN_VARIABLE_FATIMA_HOME, reader.env.GetFolderGuide().GetFatimaHome()})
	reader.appendBuiltinVar(variableValue{BUILTIN_VARIABLE_LOCAL_IPADDRESS, getDefaultIpAddress()})
	reader.appendBuiltinVar(variableValue{BUILTIN_VARIABLE_YYYYMM, ""})
	reader.appendBuiltinVar(variableValue{BUILTIN_VARIABLE_YYYYMMDD, ""})
	reader.appendBuiltinVar(variableValue{BUILTIN_VARIABLE_APP_NAME, reader.env.GetSystemProc().GetProgramName()})
	reader.appendBuiltinVar(variableValue{BUILTIN_VARIABLE_APP_FOLDER_DATA, reader.env.GetFolderGuide().GetDataFolder()})
}

// prepareMatchers serving fatima global configuration properties as variable
func (reader *PropertyPredefineReader) prepareMatchers() {
	var matchers []string

	// add builtin vars to matchers
	for _, v := range reader.builtinVariables {
		matchers = append(matchers, string(v.key))
		matchers = append(matchers, v.getValue())
	}
	builtinReplacer := strings.NewReplacer(matchers...)
	props, _ := readProperties(filepath.Join(reader.env.GetFolderGuide().GetConfFolder(), "fatima-package-predefine.properties"))

	// add package global properties to matchers
	for k, v := range props {
		keyForm := fmt.Sprintf("${%s}", k)

		// we have to call 'Replace' because package global property contains 'builtin'
		valueForm := builtinReplacer.Replace(v)
		reader.defines[keyForm] = valueForm
		matchers = append(matchers, keyForm)
		matchers = append(matchers, valueForm)
	}

	// builtinReplacer : new Replacer from a list of old, new string pairs
	reader.replacer = strings.NewReplacer(matchers...)
}

func (reader *PropertyPredefineReader) appendBuiltinVar(v variableValue) {
	reader.builtinVariables = append(reader.builtinVariables, v)
}

// getDefaultIpAddress find local ipv4 address
func getDefaultIpAddress() string {
	// func Interfaces() ([]Interface, error)
	inf, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	var min = 100
	ordered := make(map[int]string)
	for _, v := range inf {
		if !(v.Flags&net.FlagBroadcast == net.FlagBroadcast) {
			continue
		}
		if !strings.HasPrefix(v.Name, "eth") && !strings.HasPrefix(v.Name, "en") {
			continue
		}
		addrs, _ := v.Addrs()
		if len(addrs) < 1 {
			continue
		}
		var order int
		if strings.HasPrefix(v.Name, "eth") {
			order, _ = strconv.Atoi(v.Name[3:])
		} else {
			order, _ = strconv.Atoi(v.Name[2:])
		}

		for _, addr := range addrs {
			// check the address type and if it is not a loopback the display it
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ordered[order] = ipnet.IP.String()
					if order <= min {
						min = order
					}
					break
				}
			}
		}
	}

	if len(ordered) < 1 {
		return "127.0.0.1"
	}

	return ordered[min]
}
