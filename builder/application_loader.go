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
	"path/filepath"
	"strings"

	fatima "github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/crypt"
	log "github.com/fatima-go/fatima-log"
)

var configFormats = []string{"yaml", "yml", "properties"}

type configLoaderFunc func(string) (yamlLoadResult, error)

var configLoaders = map[string]configLoaderFunc{
	"yaml":       loadYamlFile,
	"yml":        loadYamlFile,
	"properties": loadPropertiesFile,
}

func loadPropertiesFile(path string) (yamlLoadResult, error) {
	values, err := readProperties(path)
	if err != nil {
		return yamlLoadResult{}, err
	}
	return yamlLoadResult{
		Values:      values,
		ListKeys:    make(map[string]bool),
		SkippedKeys: make(map[string]bool),
	}, nil
}

// LoadedApplicationConfig is the result of LoadApplicationConfig.
type LoadedApplicationConfig struct {
	Values          map[string]string
	Format          string          // "yaml" | "yml" | "properties" | "" (no file found)
	YamlListKeys    map[string]bool // keys whose original yaml value was a scalar list
	YamlSkippedKeys map[string]bool // keys skipped due to complex yaml types
}

// LoadApplicationConfig loads merged application config from appDir.
// Format search order: yaml > yml > properties.
// Base file (application.<ext>) is loaded first; profile override (application.<profile>.<ext>)
// is merged on top. Both files must use the same format.
// predefines may be nil; if provided, ${var.*} placeholders are resolved after loading.
func LoadApplicationConfig(appDir string, profile string, predefines fatima.Predefines) LoadedApplicationConfig {
	chosenExt := resolveConfigFormat(appDir, profile)
	if chosenExt == "" {
		log.Warn("cannot find application config file in %s", appDir)
		return LoadedApplicationConfig{
			Values:          make(map[string]string),
			YamlListKeys:    make(map[string]bool),
			YamlSkippedKeys: make(map[string]bool),
		}
	}

	loader := configLoaders[chosenExt]
	merged := make(map[string]string)
	listKeys := make(map[string]bool)
	skippedKeys := make(map[string]bool)

	basePath := filepath.Join(appDir, "application."+chosenExt)
	if checkFileAvailable(basePath) {
		log.Info("loading base config: %s", filepath.Base(basePath))
		if r, err := loader(basePath); err != nil {
			log.Warn("cannot load base config %s: %s", filepath.Base(basePath), err.Error())
		} else {
			mergeConfig(merged, r.Values)
			mergeMeta(listKeys, r.ListKeys)
			mergeMeta(skippedKeys, r.SkippedKeys)
		}
	}

	if profile != "" {
		overridePath := filepath.Join(appDir, fmt.Sprintf("application.%s.%s", profile, chosenExt))
		if checkFileAvailable(overridePath) {
			log.Info("applying profile override: %s", filepath.Base(overridePath))
			if r, err := loader(overridePath); err != nil {
				log.Warn("cannot load profile config %s: %s", filepath.Base(overridePath), err.Error())
			} else {
				mergeConfig(merged, r.Values)
				mergeMeta(listKeys, r.ListKeys)
				mergeMeta(skippedKeys, r.SkippedKeys)
			}
		}
	}

	for k, v := range merged {
		if predefines != nil {
			v = predefines.ResolvePredefine(v)
		}
		if strings.HasSuffix(k, SecretKeySuffix) {
			v = crypt.ResolveSecret(v)
		}
		merged[k] = v
	}

	return LoadedApplicationConfig{
		Values:          merged,
		Format:          chosenExt,
		YamlListKeys:    listKeys,
		YamlSkippedKeys: skippedKeys,
	}
}

// resolveConfigFormat determines which config format to use by checking base files first,
// then profile-only files (in yaml > yml > properties order).
func resolveConfigFormat(appDir string, profile string) string {
	for _, ext := range configFormats {
		if checkFileAvailable(filepath.Join(appDir, "application."+ext)) {
			return ext
		}
	}
	if profile != "" {
		for _, ext := range configFormats {
			if checkFileAvailable(filepath.Join(appDir, fmt.Sprintf("application.%s.%s", profile, ext))) {
				return ext
			}
		}
	}
	return ""
}

func mergeConfig(base, overlay map[string]string) {
	for k, v := range overlay {
		base[k] = v
	}
}

func mergeMeta(base, overlay map[string]bool) {
	for k, v := range overlay {
		base[k] = v
	}
}
