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
	"os"
	"path/filepath"
	"strings"
	"testing"

	fatima "github.com/fatima-go/fatima-core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPredefines is a simple Predefines implementation for testing.
type testPredefines struct {
	vars map[string]string
}

func (p *testPredefines) ResolvePredefine(value string) string {
	for k, v := range p.vars {
		value = strings.ReplaceAll(value, "${"+k+"}", v)
	}
	return value
}

func (p *testPredefines) GetDefine(key string) (string, bool) {
	v, ok := p.vars[key]
	return v, ok
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}

func TestLoadApplicationConfig(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(dir string)
		profile    string
		predefines *testPredefines
		want       map[string]string
		check      func(t *testing.T, result map[string]string)
	}{
		{
			name: "base_properties_only",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", "key1=val1\nkey2=val2\n")
			},
			want: map[string]string{"key1": "val1", "key2": "val2"},
		},
		{
			name: "base_plus_profile_override",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", "key1=base\nkey2=base2\n")
				writeTestFile(t, dir, "application.dev.properties", "key1=overridden\nkey3=new\n")
			},
			profile: "dev",
			want:    map[string]string{"key1": "overridden", "key2": "base2", "key3": "new"},
		},
		{
			name: "profile_only_no_base",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.dev.properties", "key1=val1\n")
			},
			profile: "dev",
			want:    map[string]string{"key1": "val1"},
		},
		{
			name: "yaml_base_plus_profile_override",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "db:\n  host: localhost\n  port: 5432\n")
				writeTestFile(t, dir, "application.dev.yaml", "db:\n  host: devhost\n")
			},
			profile: "dev",
			want:    map[string]string{"db.host": "devhost", "db.port": "5432"},
		},
		{
			name: "format_priority_yaml_wins",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "key1: yaml_val\n")
				writeTestFile(t, dir, "application.yml", "key1: yml_val\n")
				writeTestFile(t, dir, "application.properties", "key1=props_val\n")
			},
			want: map[string]string{"key1": "yaml_val"},
		},
		{
			name: "format_priority_yml_over_properties",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yml", "key1: yml_val\n")
				writeTestFile(t, dir, "application.properties", "key1=props_val\n")
			},
			want: map[string]string{"key1": "yml_val"},
		},
		{
			name: "format_mixing_profile_ignored",
			setup: func(dir string) {
				// base is yaml; profile override is properties → properties override ignored
				writeTestFile(t, dir, "application.yaml", "key1: base\nkey2: base2\n")
				writeTestFile(t, dir, "application.dev.properties", "key1=overridden\nkey3=new\n")
			},
			profile: "dev",
			want:    map[string]string{"key1": "base", "key2": "base2"},
		},
		{
			name: "predefine_substitution",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", "db.url=${myvar}/db\n")
			},
			predefines: &testPredefines{vars: map[string]string{"myvar": "mysql://localhost"}},
			want:       map[string]string{"db.url": "mysql://localhost/db"},
		},
		{
			name: "yaml_scalar_array_comma_join",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "hosts:\n  - a\n  - b\n  - c\n")
			},
			want: map[string]string{"hosts": "a,b,c"},
		},
		{
			name: "yaml_complex_array_skipped",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"peers:\n  - name: a\n  - name: b\nother: val\n")
			},
			check: func(t *testing.T, result map[string]string) {
				assert.Equal(t, "val", result["other"])
				_, hasPeers := result["peers"]
				assert.False(t, hasPeers, "complex array key should be skipped")
			},
		},
		{
			name:  "no_config_files_returns_empty",
			setup: func(dir string) {},
			want:  map[string]string{},
		},
		{
			name: "programname_properties_ignored",
			setup: func(dir string) {
				writeTestFile(t, dir, "myproc.properties", "should_not=load\n")
				writeTestFile(t, dir, "application.properties", "key1=val1\n")
			},
			want: map[string]string{"key1": "val1"},
		},
		{
			name: "yaml_dot_key_conflict_no_panic",
			setup: func(dir string) {
				// "a.b" as a literal key and nested a.b both flatten to "a.b"
				writeTestFile(t, dir, "application.yaml", "\"a.b\": literal\na:\n  b: nested\n")
			},
			check: func(t *testing.T, result map[string]string) {
				v, exists := result["a.b"]
				assert.True(t, exists, "a.b key should exist after conflict resolution")
				assert.True(t, v == "literal" || v == "nested", "value should be one of the two conflicting values, got: %s", v)
			},
		},
		{
			name: "properties_trimspace",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", " key1 = val1 \nkey2= val2\n")
			},
			want: map[string]string{"key1": "val1", "key2": "val2"},
		},
		{
			name: "yaml_nested_three_levels",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "a:\n  b:\n    c: deep\n")
			},
			want: map[string]string{"a.b.c": "deep"},
		},
		{
			name: "yaml_predefine_substitution",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "db:\n  url: ${host}/mydb\n")
			},
			predefines: &testPredefines{vars: map[string]string{"host": "mysql://localhost"}},
			want:       map[string]string{"db.url": "mysql://localhost/mydb"},
		},
		{
			name: "yaml_integer_value",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "server:\n  port: 8080\n")
			},
			want: map[string]string{"server.port": "8080"},
		},
		{
			name: "yaml_boolean_value",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "feature:\n  enabled: true\n")
			},
			want: map[string]string{"feature.enabled": "true"},
		},
		// --- multi-doc 케이스 ---
		{
			name: "yaml_multidoc_base_plus_profile_dev",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"servicedb:\n  url: base_url\n  maxidleconns: 0\n  maxopenconns: 4\n"+
						"---\nfatima:\n  profile: dev\nservicedb:\n  maxidleconns: 4\n"+
						"---\nfatima:\n  profile: qa\nservicedb:\n  maxidleconns: 12\n")
			},
			profile: "dev",
			want: map[string]string{
				"servicedb.url":          "base_url",
				"servicedb.maxidleconns": "4",
				"servicedb.maxopenconns": "4",
			},
		},
		{
			name: "yaml_multidoc_no_matching_profile",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"key1: base\n"+
						"---\nfatima:\n  profile: dev\nkey1: devval\n"+
						"---\nfatima:\n  profile: qa\nkey1: qaval\n")
			},
			profile: "stg",
			want:    map[string]string{"key1": "base"},
		},
		{
			name: "yaml_multidoc_empty_profile",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"key1: base\n"+
						"---\nfatima:\n  profile: dev\nkey1: devval\n")
			},
			profile: "",
			want:    map[string]string{"key1": "base"},
		},
		{
			name: "yaml_multidoc_overrides_separate_file",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"key1: base\n"+
						"---\nfatima:\n  profile: dev\nkey1: multidoc_dev\n")
				// 별도파일이 존재해도 multi-doc이면 무시되어야 한다
				writeTestFile(t, dir, "application.dev.yaml", "key1: separate_dev\n")
			},
			profile: "dev",
			want:    map[string]string{"key1": "multidoc_dev"},
		},
		{
			name: "yaml_multidoc_duplicate_profile_warn",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"key1: base\n"+
						"---\nfatima:\n  profile: dev\nkey1: first_dev\n"+
						"---\nfatima:\n  profile: dev\nkey1: second_dev\n")
			},
			profile: "dev",
			want:    map[string]string{"key1": "second_dev"},
		},
		{
			name: "yaml_multidoc_fatima_profile_key_removed",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"key1: base\n"+
						"---\nfatima:\n  profile: dev\nkey2: devval\n")
			},
			profile: "dev",
			check: func(t *testing.T, result map[string]string) {
				_, hasFatimaProfile := result["fatima.profile"]
				assert.False(t, hasFatimaProfile, "fatima.profile should be stripped from result")
				assert.Equal(t, "base", result["key1"])
				assert.Equal(t, "devval", result["key2"])
			},
		},
		{
			name: "yml_multidoc_supported",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yml",
					"key1: base\n"+
						"---\nfatima:\n  profile: dev\nkey1: devval\n")
			},
			profile: "dev",
			want:    map[string]string{"key1": "devval"},
		},
		{
			name: "yaml_single_doc_unchanged",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "key1: base\nkey2: val2\n")
				writeTestFile(t, dir, "application.dev.yaml", "key1: overridden\n")
			},
			profile: "dev",
			want:    map[string]string{"key1": "overridden", "key2": "val2"},
		},
		{
			name: "yaml_multidoc_base_only",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml",
					"key1: base\n"+
						"---\nfatima:\n  profile: dev\nkey1: devval\n")
			},
			profile: "qa",
			want:    map[string]string{"key1": "base"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			// pass nil interface explicitly to avoid non-nil interface wrapping a nil pointer
			var predefines fatima.Predefines
			if tt.predefines != nil {
				predefines = tt.predefines
			}

			loaded := LoadApplicationConfig(dir, tt.profile, predefines)

			if tt.check != nil {
				tt.check(t, loaded.Values)
			} else {
				assert.Equal(t, tt.want, loaded.Values)
			}
		})
	}
}

func newReaderFromDir(t *testing.T, dir string, profile string) *PropertyConfigReader {
	t.Helper()
	loaded := LoadApplicationConfig(dir, profile, nil)
	return &PropertyConfigReader{
		configuration:   loaded.Values,
		format:          loaded.Format,
		yamlListKeys:    loaded.YamlListKeys,
		yamlSkippedKeys: loaded.YamlSkippedKeys,
	}
}

func TestPropertyConfigReader_GetList(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(dir string)
		profile string
		key     string
		want    []string
		wantErr bool
	}{
		{
			name: "properties_single_value",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", "key=a\n")
			},
			key:  "key",
			want: []string{"a"},
		},
		{
			name: "properties_comma_separated",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", "key=a,b, c\n")
			},
			key:  "key",
			want: []string{"a", "b", "c"},
		},
		{
			name: "properties_empty_value",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", "key=\n")
			},
			key:  "key",
			want: []string{},
		},
		{
			name: "properties_not_found",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.properties", "other=val\n")
			},
			key:     "key",
			wantErr: true,
		},
		{
			name: "yaml_scalar_list",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "key:\n  - a\n  - b\n  - c\n")
			},
			key:  "key",
			want: []string{"a", "b", "c"},
		},
		{
			name: "yaml_numeric_list",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "key:\n  - 1\n  - 2\n  - 3\n")
			},
			key:  "key",
			want: []string{"1", "2", "3"},
		},
		{
			name: "yaml_single_scalar",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "key: hello\n")
			},
			key:  "key",
			want: []string{"hello"},
		},
		{
			name: "yaml_single_scalar_with_comma",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "key: \"a,b\"\n")
			},
			key:  "key",
			want: []string{"a,b"},
		},
		{
			name: "yaml_empty_value",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "key: \"\"\n")
			},
			key:  "key",
			want: []string{},
		},
		{
			name: "yaml_not_found",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "other: val\n")
			},
			key:     "key",
			wantErr: true,
		},
		{
			name: "yaml_complex_array_skipped",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "peers:\n  - name: a\n  - name: b\nother: val\n")
			},
			key:     "peers",
			wantErr: true,
		},
		{
			name: "yaml_nested_list",
			setup: func(dir string) {
				writeTestFile(t, dir, "application.yaml", "openapi:\n  domain:\n    list:\n      - curation\n      - display/api-docs/display\n      - display/api-docs/promotion\n")
			},
			key:  "openapi.domain.list",
			want: []string{"curation", "display/api-docs/display", "display/api-docs/promotion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			reader := newReaderFromDir(t, dir, tt.profile)
			got, err := reader.GetList(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
