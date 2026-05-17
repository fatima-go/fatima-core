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
 */

package builder

import (
	"fmt"
	"os"
	"strings"

	log "github.com/fatima-go/fatima-log"
	"gopkg.in/yaml.v3"
)

type yamlLoadResult struct {
	Values      map[string]string
	ListKeys    map[string]bool
	SkippedKeys map[string]bool
}

func loadYamlFile(path string) (yamlLoadResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return yamlLoadResult{}, err
	}
	var raw map[string]any
	if err = yaml.Unmarshal(data, &raw); err != nil {
		return yamlLoadResult{}, err
	}
	result := yamlLoadResult{
		Values:      make(map[string]string),
		ListKeys:    make(map[string]bool),
		SkippedKeys: make(map[string]bool),
	}
	flattenYaml("", raw, &result)
	return result, nil
}

func flattenYaml(prefix string, in map[string]any, result *yamlLoadResult) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			flattenYaml(key, t, result)
		case []any:
			flattenYamlSlice(key, t, result)
		case nil:
			// skip
		default:
			if _, exists := result.Values[key]; exists {
				log.Warn("yaml key conflict at '%s', overwriting previous value", key)
			}
			result.Values[key] = fmt.Sprintf("%v", t)
		}
	}
}

// flattenYamlSlice joins scalar list elements with comma.
// Records the key in ListKeys on success, or SkippedKeys if any element is a complex type.
func flattenYamlSlice(key string, arr []any, result *yamlLoadResult) {
	parts := make([]string, 0, len(arr))
	for _, elem := range arr {
		switch elem.(type) {
		case map[string]any, []any:
			log.Warn("yaml array of complex types unsupported at key '%s', skipped", key)
			result.SkippedKeys[key] = true
			return
		case nil:
			// skip nil elements
		default:
			parts = append(parts, fmt.Sprintf("%v", elem))
		}
	}
	if len(parts) > 0 {
		result.Values[key] = strings.Join(parts, ",")
		result.ListKeys[key] = true
	}
}
