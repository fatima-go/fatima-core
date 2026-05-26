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
	"strings"

	log "github.com/fatima-go/fatima-log"
	"gopkg.in/yaml.v3"
)

type yamlLoadResult struct {
	Values      map[string]string
	ListKeys    map[string]bool
	SkippedKeys map[string]bool
	IsMultiDoc  bool
}

func newYamlLoadResult() yamlLoadResult {
	return yamlLoadResult{
		Values:      make(map[string]string),
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
		var doc map[string]any
		if err = dec.Decode(&doc); err == io.EOF {
			break
		} else if err != nil {
			return yamlLoadResult{}, err
		}
		if doc != nil {
			docs = append(docs, doc)
		}
	}

	if len(docs) <= 1 {
		if len(docs) == 1 {
			return flattenDoc(docs[0]), nil
		}
		return newYamlLoadResult(), nil
	}

	// multi-doc 모드: fatima.profile 키 유무로 base/profile 블록 분류
	result := newYamlLoadResult()
	result.IsMultiDoc = true

	base, matching := classifyDocs(docs, profile, path)
	if base != nil {
		mergeFlattened(&result, flattenDoc(base), "")
	}
	for i, m := range matching {
		if i > 0 {
			log.Warn("duplicate profile '%s' block in %s, later overrides earlier", profile, path)
		}
		mergeFlattened(&result, flattenDoc(m), fmt.Sprintf("profile '%s'", profile))
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

// flattenDoc flattens a single yaml document into a fresh yamlLoadResult.
// Intra-document key conflicts (e.g. dot-notation key vs nested map) are logged as WARN.
func flattenDoc(doc map[string]any) yamlLoadResult {
	r := newYamlLoadResult()
	flattenYaml("", doc, &r)
	return r
}

// mergeFlattened merges src into dst. For each key in src whose value differs from
// the existing value in dst, logs INFO with sourceLabel before overwriting.
// Pass an empty sourceLabel when merging a base with no prior content (no overrides expected).
func mergeFlattened(dst *yamlLoadResult, src yamlLoadResult, sourceLabel string) {
	for k, v := range src.Values {
		if existing, exists := dst.Values[k]; exists && existing != v && sourceLabel != "" {
			log.Info("config key '%s' overridden by %s (was '%s', now '%s')", k, sourceLabel, existing, v)
		}
		dst.Values[k] = v
	}
	for k := range src.ListKeys {
		dst.ListKeys[k] = true
	}
	for k := range src.SkippedKeys {
		dst.SkippedKeys[k] = true
	}
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
