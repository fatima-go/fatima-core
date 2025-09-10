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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/crypt"
	"github.com/fatima-go/fatima-log"
)

type PropertyConfigReader struct {
	predefines    fatima.Predefines
	env           fatima.FatimaEnv
	configuration map[string]string
}

// NewPropertyConfigReader serving properties(key/value) for process
func NewPropertyConfigReader(env fatima.FatimaEnv, predefines fatima.Predefines) *PropertyConfigReader {
	instance := new(PropertyConfigReader)
	instance.env = env

	// predefines : builtin + fatima global config
	instance.predefines = predefines
	instance.configuration = make(map[string]string)

	// find proper properties file path for process
	propFilePath := loadApplicationProperty(env)
	if len(propFilePath) == 0 {
		log.Warn("cannot load properties file...")
		return instance
	}

	log.Info("using properties file : %s", filepath.Base(propFilePath))

	// read properties (key=value pairs)
	props, err := readProperties(propFilePath)
	if err != nil {
		log.Warn("cannot load properties file : %s", err.Error())
	}
	if props != nil {
		for k, v := range props {
			// from application.xxx.properties
			// e.g) writedb.url=${var.db.write.url}?autocommit=true&timeout=180s&readTimeout=180s
			// k : writedb.url
			// v : ${var.db.write.url}?autocommit=true&timeout=180s&readTimeout=180s
			// maybe fatima global predefine file (fatima-package-predefines.properties) contains "var.db.write.url"
			// so we need to resolve(replace)
			instance.configuration[k] = predefines.ResolvePredefine(v) // PropertyPredefineReader.ResolvePredefine()

			// need secret replace
			if strings.HasSuffix(k, SecretKeySuffix) {
				// overwrite with resolved content1
				instance.configuration[k] = crypt.ResolveSecret(v)
			}
		}
	}
	return instance
}

// loadApplicationProperty find proper properties file path for process
func loadApplicationProperty(env fatima.FatimaEnv) string {
	list := make([]string, 0)
	if env.GetProfile() != "" {
		list = append(list, fmt.Sprintf("%s.%s.properties", env.GetSystemProc().GetProgramName(), env.GetProfile()))
		list = append(list, fmt.Sprintf("application.%s.properties", env.GetProfile()))
	}

	list = append(list, fmt.Sprintf("%s.properties", env.GetSystemProc().GetProgramName()))
	list = append(list, "application.properties")

	for _, v := range list {
		propFilePath := filepath.Join(env.GetFolderGuide().GetAppFolder(), v)
		if checkFileAvailable(propFilePath) {
			return propFilePath
		}
	}

	return ""
}

func (this *PropertyConfigReader) GetValue(key string) (string, bool) {
	v, ok := this.configuration[key]
	return v, ok
}

func (this *PropertyConfigReader) GetString(key string) (string, error) {
	v, ok := this.configuration[key]
	if !ok {
		return "", fmt.Errorf("not found key in config : %s", key)
	}
	return v, nil
}

func (this *PropertyConfigReader) GetInt(key string) (int, error) {
	v, ok := this.configuration[key]
	if !ok {
		return 0, fmt.Errorf("not found key in config : %s", key)
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("not numeric value for key %s : %s", key, err.Error())
	}

	return i, nil
}

func (this *PropertyConfigReader) GetBool(key string) (bool, error) {
	v, ok := this.configuration[key]
	if !ok {
		return false, fmt.Errorf("not found key in config : %s", key)
	}

	switch strings.ToUpper(v) {
	case "TRUE":
		return true, nil
	}

	return false, nil
}

func (this *PropertyConfigReader) ResolvePredefine(value string) string {
	return this.predefines.ResolvePredefine(value)
}

func (this *PropertyConfigReader) GetDefine(key string) (string, bool) {
	return this.predefines.GetDefine(key)
}

func checkFileAvailable(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Trace("file [%s] does not exist", filepath.Base(path))
		return false
	}

	return true
}
