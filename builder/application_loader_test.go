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

			result := LoadApplicationConfig(dir, tt.profile, predefines)

			if tt.check != nil {
				tt.check(t, result)
			} else {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}
