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

func loadYamlFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err = yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	result := make(map[string]string)
	flattenYaml("", raw, result)
	return result, nil
}

func flattenYaml(prefix string, in map[string]any, out map[string]string) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			flattenYaml(key, t, out)
		case []any:
			flattenYamlSlice(key, t, out)
		case nil:
			// skip
		default:
			if _, exists := out[key]; exists {
				log.Warn("yaml key conflict at '%s', overwriting previous value", key)
			}
			out[key] = fmt.Sprintf("%v", t)
		}
	}
}

// flattenYamlSlice joins scalar list elements with comma.
// Skips the entire key if any element is a complex type (map or nested slice).
func flattenYamlSlice(key string, arr []any, out map[string]string) {
	parts := make([]string, 0, len(arr))
	for _, elem := range arr {
		switch elem.(type) {
		case map[string]any, []any:
			log.Warn("yaml array of complex types unsupported at key '%s', skipped", key)
			return
		case nil:
			// skip nil elements
		default:
			parts = append(parts, fmt.Sprintf("%v", elem))
		}
	}
	if len(parts) > 0 {
		out[key] = strings.Join(parts, ",")
	}
}
