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
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	log "github.com/fatima-go/fatima-log"
	"gopkg.in/yaml.v3"
)

type yamlLoadResult struct {
	Values      map[string]string
	ListKeys    map[string]bool
	SkippedKeys map[string]bool
	IsMultiDoc  bool
	Nodes       map[string]any
	ComplexKeys map[string]bool
}

func newYamlLoadResult() yamlLoadResult {
	return yamlLoadResult{
		Values:      make(map[string]string),
		Nodes:       make(map[string]any),
		ComplexKeys: make(map[string]bool),
		ListKeys:    make(map[string]bool),
		SkippedKeys: make(map[string]bool),
	}
}

func loadYamlFile(path string, profile string) (yamlLoadResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return yamlLoadResult{}, err
	}

	docs := make([]map[string]any, 0)
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var raw any
		if err = dec.Decode(&raw); err == io.EOF {
			break
		} else if err != nil {
			return yamlLoadResult{}, fmt.Errorf("invalid YAML configuration")
		}
		if raw != nil {
			doc, ok := raw.(map[string]any)
			if !ok {
				return yamlLoadResult{}, fmt.Errorf("YAML configuration requires a string-keyed object root")
			}
			if err := validateConfigNode(doc); err != nil {
				return yamlLoadResult{}, err
			}
			docs = append(docs, doc)
		}
	}

	if len(docs) <= 1 {
		if len(docs) == 1 {
			return yamlDocument(docs[0]), nil
		}
		return newYamlLoadResult(), nil
	}

	// multi-doc 모드: fatima.profile 키 유무로 base/profile 블록 분류
	result := newYamlLoadResult()
	result.IsMultiDoc = true

	base, matching := classifyDocs(docs, profile, path)
	if base != nil {
		mergeLoaded(&result, yamlDocument(base), "")
	}
	for i, m := range matching {
		if i > 0 {
			log.Warn("duplicate profile '%s' block in %s, later overrides earlier", profile, path)
		}
		mergeLoaded(&result, yamlDocument(m), fmt.Sprintf("profile '%s'", profile))
	}
	return result, nil
}

// classifyDocs separates documents into a base (no fatima.profile) and matching profile docs.
func classifyDocs(docs []map[string]any, profile string, path string) (base map[string]any, matching []map[string]any) {
	matching = make([]map[string]any, 0)
	for _, d := range docs {
		p := extractAndStripProfile(d)
		switch {
		case p == "":
			if base != nil {
				log.Warn("multiple base documents found in %s, last one wins", path)
			}
			base = d
		case p == profile && profile != "":
			matching = append(matching, d)
		}
	}
	return
}

// extractAndStripProfile reads fatima.profile from the document, removes the key,
// and cleans up the fatima map if it becomes empty. Returns the profile string (empty if absent).
func extractAndStripProfile(doc map[string]any) string {
	fatimaVal, ok := doc["fatima"]
	if !ok {
		return ""
	}
	fatimaMap, ok := fatimaVal.(map[string]any)
	if !ok {
		return ""
	}
	profileVal, ok := fatimaMap["profile"]
	if !ok {
		return ""
	}
	profile := fmt.Sprintf("%v", profileVal)
	delete(fatimaMap, "profile")
	if len(fatimaMap) == 0 {
		delete(doc, "fatima")
	}
	return profile
}

// yamlDocument retains the YAML types until all selected documents are merged.
func yamlDocument(doc map[string]any) yamlLoadResult {
	r := newYamlLoadResult()
	r.Nodes = doc
	return r
}

func mergeLoaded(dst *yamlLoadResult, src yamlLoadResult, label string) {
	mergeConfigMaps(dst.Nodes, src.Nodes, "", label)
}

func mergeConfigMaps(dst, src map[string]any, prefix, label string) {
	for _, k := range sortedConfigKeys(src) {
		v := src[k]
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		previous, exists := dst[k]
		oldMap, oldOK := previous.(map[string]any)
		newMap, newOK := v.(map[string]any)
		if oldOK && newOK {
			mergeConfigMaps(oldMap, newMap, key, label)
		} else {
			if exists && !reflect.DeepEqual(previous, v) && label != "" {
				log.Info("config key '%s' overridden by %s", key, label)
			}
			dst[k] = cloneConfigNode(v)
		}
	}
}

func sortedConfigKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Non-string YAML map keys have no representation in the public path API.
func validateConfigNode(v any) error {
	switch n := v.(type) {
	case map[any]any:
		return fmt.Errorf("YAML configuration requires string map keys")
	case map[string]any:
		for _, k := range sortedConfigKeys(n) {
			if err := validateConfigNode(n[k]); err != nil {
				return err
			}
		}
	case []any:
		for _, value := range n {
			if err := validateConfigNode(value); err != nil {
				return err
			}
		}
	}
	return nil
}

// Project after merging/resolving. Sorted traversal makes literal dotted keys
// win over equivalent nested paths, independent of Go map iteration order.
func flattenYaml(prefix string, in map[string]any, result *yamlLoadResult) {
	for _, k := range sortedConfigKeys(in) {
		v := in[k]
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if m, ok := v.(map[string]any); ok {
			flattenYaml(key, m, result)
			continue
		}
		if _, exists := result.Values[key]; exists || result.ComplexKeys[key] {
			log.Warn("yaml key conflict at '%s', overwriting previous value", key)
		}
		delete(result.Values, key)
		delete(result.ListKeys, key)
		delete(result.ComplexKeys, key)
		switch n := v.(type) {
		case []any:
			parts := make([]string, 0, len(n))
			complex := false
			for _, elem := range n {
				switch elem.(type) {
				case map[string]any, []any:
					complex = true
				case nil:
				default:
					parts = append(parts, fmt.Sprint(elem))
				}
			}
			if complex {
				result.ComplexKeys[key] = true
				continue
			}
			result.Values[key] = strings.Join(parts, ",")
			result.ListKeys[key] = true
		case nil: // Explicit null is retained in Nodes but has no string projection.
		default:
			result.Values[key] = fmt.Sprint(n)
		}
	}
}
