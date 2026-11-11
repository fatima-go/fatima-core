package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	fatima "github.com/fatima-go/fatima-core"
	log "github.com/fatima-go/fatima-log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureDir(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir(filepath.Join("testdata", "structured-config", name))
	require.NoError(t, err)
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("testdata", "structured-config", name, entry.Name()))
		require.NoError(t, err)
		writeTestFile(t, dir, entry.Name(), string(data))
	}
	return dir
}

func checkNode(t *testing.T, c fatima.StructuredConfig, want any, path ...string) {
	t.Helper()
	got, exists := c.LookupNode(path...)
	require.True(t, exists, "path %q", path)
	assert.Equal(t, want, got, "path %q", path)
}

func TestStructuredSeparateFixture(t *testing.T) {
	dir := fixtureDir(t, "separate")
	predefines := &testPredefines{vars: map[string]string{
		"var.fixture.cache.host": "cache.qa.example.test", "var.fixture.token": "b64:Zml4dHVyZS10b2tlbg==",
	}}
	loaded := LoadApplicationConfig(dir, "qa", predefines)
	c := loaded.Structured
	require.Empty(t, loaded.YamlSkippedKeys)
	checkNode(t, c, 18080, "server", "port")
	checkNode(t, c, true, "server", "enabled")
	checkNode(t, c, []any{"qa"}, "server", "labels")
	checkNode(t, c, "left,right", "server", "caption")
	checkNode(t, c, "https://git.example.test", "services", "git", "base")
	checkNode(t, c, []any{map[string]any{"name": "qa-storage-lib", "repository": "demo/qa-storage-lib"}}, "services", "references")
	checkNode(t, c, []any{}, "services", "exclude")
	checkNode(t, c, []any{map[string]any{"name": "qa-order-api", "repository": "demo/qa-order-api", "priority": 1, "tags": []any{"qa"}, "paths": []any{"src/**"}}}, "services", "group", "api")
	checkNode(t, c, []any{map[string]any{"name": "batch-worker", "repository": "demo/batch-worker"}}, "services", "group", "backend")
	type connector struct {
		Name    string
		Enabled bool
		Pool    struct{ Size int }
		Auth    struct {
			Token string `yaml:"token.secret"`
		}
		Nodes []struct {
			Host string
			Port int
		}
	}
	var connectors []connector
	require.NoError(t, c.DecodeNode(&connectors, "connectors"))
	require.Len(t, connectors, 1)
	assert.Equal(t, "cache", connectors[0].Name)
	assert.True(t, connectors[0].Enabled)
	assert.Equal(t, 4, connectors[0].Pool.Size)
	assert.Equal(t, "fixture-token", connectors[0].Auth.Token)
	require.Len(t, connectors[0].Nodes, 1)
	assert.Equal(t, "cache.qa.example.test", connectors[0].Nodes[0].Host)
	assert.Equal(t, 6379, connectors[0].Nodes[0].Port)
	assert.Equal(t, "18080", loaded.Values["server.port"])
	assert.Equal(t, "", loaded.Values["services.exclude"])
	for _, profile := range []string{"", "missing"} {
		base := LoadApplicationConfig(dir, profile, predefines).Structured
		checkNode(t, base, 8080, "server", "port")
		var refs []map[string]any
		require.NoError(t, base.DecodeNode(&refs, "services", "references"))
		assert.Len(t, refs, 2)
	}
}

func TestStructuredMultidocFixture(t *testing.T) {
	dir := fixtureDir(t, "multidoc")
	for _, tt := range []struct {
		profile string
		port    int
		name    string
	}{
		{"qa", 18080, "qa-final-api"}, {"prod", 28080, "prod-api"}, {"stg", 8080, "base-api"}, {"", 8080, "base-api"},
	} {
		t.Run(tt.profile, func(t *testing.T) {
			loaded := LoadApplicationConfig(dir, tt.profile, nil)
			checkNode(t, loaded.Structured, tt.port, "server", "port")
			var apis []struct{ Name string }
			require.NoError(t, loaded.Structured.DecodeNode(&apis, "services", "group", "api"))
			require.Len(t, apis, 1)
			assert.Equal(t, tt.name, apis[0].Name)
			_, exists := loaded.Structured.LookupNode("fatima", "profile")
			assert.False(t, exists)
			assert.NotContains(t, loaded.Values, "fatima.profile")
		})
	}
}

func TestStructuredEdgeFixture(t *testing.T) {
	dir := fixtureDir(t, "edge")
	c := LoadApplicationConfig(dir, "qa", nil).Structured
	for _, tt := range []struct {
		key   string
		value any
	}{
		{"shrink", []any{map[string]any{"name": "only"}}},
		{"clear", []any{}}, {"inherit", []any{map[string]any{"name": "keep"}}},
		{"nullable", nil}, {"switch", []any{"new"}},
		{"nested", []any{[]any{1, 2}, []any{3}}},
		{"mixed", []any{map[string]any{"name": "object"}, "text", 7, true, nil}},
		{"emptyObject", map[string]any{}}, {"values", []any{"a,b", "c", "", "  d  "}},
	} {
		t.Run(tt.key, func(t *testing.T) { checkNode(t, c, tt.value, "cases", tt.key) })
	}
	for _, path := range [][]string{{"cases", "missing"}, {"cases", "switch", "old", "enabled"}} {
		_, ok := c.LookupNode(path...)
		assert.False(t, ok)
	}
	reader := newReaderFromDir(t, dir, "qa")
	list, err := reader.GetList("cases.clear")
	require.NoError(t, err)
	assert.Equal(t, []string{}, list)
	list, err = reader.GetList("cases.values")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c", "d"}, list)
	_, ok := reader.GetValue("cases.nullable")
	assert.False(t, ok)
	_, ok = reader.GetValue("cases.switch.old.enabled")
	assert.False(t, ok)
	_, err = reader.GetList("cases.mixed")
	require.ErrorContains(t, err, "unsupported value type")
}

func TestStructuredDecodeAndIsolation(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "application.yaml", `value: "private-fixture-value"
number: 300
flag: true
fraction: 1.5
negative: -1
large: 18446744073709551615
nullable: null
empty: []
items: [{name: first, tags: [one]}]
"a.b": literal
a: {b: nested}
"": empty-key
`)
	c := LoadApplicationConfig(dir, "", nil).Structured
	for _, tt := range []struct {
		name  string
		path  []string
		dst   any
		cause error
	}{
		{"string not int", []string{"value"}, new(int), fatima.ErrConfigTypeMismatch},
		{"int not string", []string{"number"}, new(string), fatima.ErrConfigTypeMismatch},
		{"overflow", []string{"number"}, new(int8), fatima.ErrConfigTypeMismatch},
		{"signed overflow", []string{"large"}, new(int64), fatima.ErrConfigTypeMismatch},
		{"negative unsigned", []string{"negative"}, new(uint), fatima.ErrConfigTypeMismatch},
		{"int not float", []string{"number"}, new(float64), fatima.ErrConfigTypeMismatch},
		{"float not int", []string{"fraction"}, new(int), fatima.ErrConfigTypeMismatch},
		{"null scalar", []string{"nullable"}, new(bool), fatima.ErrConfigTypeMismatch},
		{"missing", []string{"absent"}, new(int), fatima.ErrConfigNodeNotFound},
		{"nil", nil, nil, fatima.ErrInvalidConfigTarget},
		{"typed nil", nil, (*int)(nil), fatima.ErrInvalidConfigTarget},
		{"not pointer", nil, 1, fatima.ErrInvalidConfigTarget},
		{"mixed target", []string{"items"}, new([]string), fatima.ErrConfigTypeMismatch},
		{"array length", []string{"items"}, new([2]map[string]any), fatima.ErrConfigTypeMismatch},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := c.DecodeNode(tt.dst, tt.path...)
			require.ErrorIs(t, err, tt.cause)
			assert.NotContains(t, err.Error(), "private-fixture-value")
			var detail *fatima.ConfigNodeError
			require.ErrorAs(t, err, &detail)
			assert.NotEmpty(t, detail.Expected)
		})
	}
	var n int64
	require.NoError(t, c.DecodeNode(&n, "number"))
	assert.EqualValues(t, 300, n)
	var u uint64
	require.NoError(t, c.DecodeNode(&u, "large"))
	assert.Equal(t, ^uint64(0), u)
	var fraction float64
	require.NoError(t, c.DecodeNode(&fraction, "fraction"))
	assert.Equal(t, 1.5, fraction)
	ptr := new(int)
	require.NoError(t, c.DecodeNode(&ptr, "nullable"))
	assert.Nil(t, ptr)
	var empty []string
	require.NoError(t, c.DecodeNode(&empty, "empty"))
	assert.NotNil(t, empty)
	assert.Empty(t, empty)
	checkNode(t, c, "literal", "a.b")
	checkNode(t, c, "nested", "a", "b")
	checkNode(t, c, "empty-key", "")
	_, ok := c.LookupNode("items", "0")
	assert.False(t, ok)
	root, _ := c.LookupNode()
	root.(map[string]any)["items"].([]any)[0].(map[string]any)["name"] = "changed"
	var decoded any
	require.NoError(t, c.DecodeNode(&decoded, "items"))
	decoded.([]any)[0].(map[string]any)["tags"].([]any)[0] = "changed"
	checkNode(t, c, []any{map[string]any{"name": "first", "tags": []any{"one"}}}, "items")
	before := struct {
		Number int
		Value  int
	}{Number: 42, Value: 43}
	require.Error(t, c.DecodeNode(&before))
	assert.Equal(t, 42, before.Number)
	assert.Equal(t, 43, before.Value)
}

func TestStructuredProjectionAndSelection(t *testing.T) {
	for _, tt := range []struct {
		name, base, override string
		want                 map[string]string
	}{
		{"clear", "list: [a, b]\n", "list: []\n", map[string]string{"list": ""}},
		{"inherit", "list: [a, b]\n", "other: x\n", map[string]string{"list": "a,b", "other": "x"}},
		{"null", "value: old\n", "value: null\n", map[string]string{}},
		{"map to list", "value: {old: true}\n", "value: [new]\n", map[string]string{"value": "new"}},
		{"list to map", "value: [old]\n", "value: {new: true}\n", map[string]string{"value.new": "true"}},
		{"list to scalar", "value: [old]\n", "value: 'a,b'\n", map[string]string{"value": "a,b"}},
		{"complex to simple", "value: [{name: old}]\n", "value: [new]\n", map[string]string{"value": "new"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTestFile(t, dir, "application.yaml", tt.base)
			writeTestFile(t, dir, "application.qa.yaml", tt.override)
			loaded := LoadApplicationConfig(dir, "qa", nil)
			assert.Equal(t, tt.want, loaded.Values)
			if tt.name == "list to scalar" {
				list, err := newReaderFromDir(t, dir, "qa").GetList("value")
				require.NoError(t, err)
				assert.Equal(t, []string{"a,b"}, list)
			}
		})
	}
	dir := t.TempDir()
	writeTestFile(t, dir, "application.qa.yml", "items: [{name: only}]\n")
	c := LoadApplicationConfig(dir, "qa", nil)
	assert.Equal(t, "yml", c.Format)
	checkNode(t, c.Structured, []any{map[string]any{"name": "only"}}, "items")
	writeTestFile(t, dir, "application.yaml", "a: {b: nested}\n'a.b': literal\n")
	for i := 0; i < 15; i++ {
		c := LoadApplicationConfig(dir, "", nil)
		assert.Equal(t, "literal", c.Values["a.b"])
	}
	writeTestFile(t, dir, "application.yaml", "fatima: {profile: qa}\nvalue: single\n")
	c = LoadApplicationConfig(dir, "", nil)
	checkNode(t, c.Structured, "qa", "fatima", "profile")
	writeTestFile(t, dir, "application.yaml", "old: removed\n---\nbase: last\n---\nfatima: {profile: qa}\nextra: first\n---\nfatima: {profile: qa}\nextra: final\n")
	c = LoadApplicationConfig(dir, "qa", nil)
	assert.Equal(t, map[string]string{"base": "last", "extra": "final"}, c.Values)
}

func TestStructuredStringsAndProperties(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "application.yaml", `top.secret: b64:YjY0OmRHVnpkQT09
items:
 - token.secret: "${token}"
   values: ["${bool}", "${number}", "${list}", "${missing}"]
secrets:
 secret: [b64:YQ==, b64:Yg==]
broken.secret: b64:invalid!
`)
	pre := &testPredefines{vars: map[string]string{"token": "b64:Zml4dHVyZS10b2tlbg==", "bool": "false", "number": "123", "list": "[a,b]"}}
	c := LoadApplicationConfig(dir, "", pre)
	checkNode(t, c.Structured, "b64:dGVzdA==", "top.secret") // One decode only.
	assert.Equal(t, "b64:dGVzdA==", c.Values["top.secret"])
	checkNode(t, c.Structured, []any{"a", "b"}, "secrets", "secret")
	checkNode(t, c.Structured, "invalid!", "broken.secret") // Existing failure return.
	var items []map[string]any
	require.NoError(t, c.Structured.DecodeNode(&items, "items"))
	require.Len(t, items, 1)
	assert.Equal(t, "fixture-token", items[0]["token.secret"])
	assert.Equal(t, []any{"false", "123", "[a,b]", "${missing}"}, items[0]["values"])
	props := t.TempDir()
	writeTestFile(t, props, "application.properties", "a.b=123\ntoken.secret=${token}\n")
	pc := LoadApplicationConfig(props, "", pre)
	checkNode(t, pc.Structured, "123", "a.b")
	_, ok := pc.Structured.LookupNode("a", "b")
	assert.False(t, ok)
	assert.Equal(t, "fixture-token", pc.Values["token.secret"])
}

func TestInvalidStructuredYaml(t *testing.T) {
	for _, input := range []string{"1: private-fixture-value\n", "- private-fixture-value\n", "key: one\nkey: private-fixture-value\n", "key: [private-fixture-value", "items: [{1: private-fixture-value}]", "items: [*missing]"} {
		dir := t.TempDir()
		writeTestFile(t, dir, "application.yaml", input)
		_, err := loadYamlFile(filepath.Join(dir, "application.yaml"), "")
		require.Error(t, err)
		assert.False(t, strings.Contains(err.Error(), "private-fixture-value"))
		// The public compatibility loader still warns and continues on file errors.
		loaded := LoadApplicationConfig(dir, "", nil)
		assert.Empty(t, loaded.Values)
	}
}

// This implementation deliberately declares only the five original methods.
type legacyConfig struct{}

func (legacyConfig) GetValue(string) (string, bool)   { return "", false }
func (legacyConfig) GetString(string) (string, error) { return "", nil }
func (legacyConfig) GetInt(string) (int, error)       { return 0, nil }
func (legacyConfig) GetBool(string) (bool, error)     { return false, nil }
func (legacyConfig) GetList(string) ([]string, error) { return nil, nil }

var _ fatima.Config = legacyConfig{}

type configTestEnv struct {
	fatima.FatimaEnv
	dir string
}

func (e configTestEnv) GetProfile() string                 { return "qa" }
func (e configTestEnv) GetFolderGuide() fatima.FolderGuide { return configTestFolders{dir: e.dir} }

type configTestFolders struct {
	fatima.FolderGuide
	dir string
}

func (f configTestFolders) GetAppFolder() string { return f.dir }

type configTestBuilder struct {
	FatimaRuntimeBuilder
	config fatima.Config
}

func (b configTestBuilder) GetConfig() fatima.Config { return b.config }

func TestStructuredRuntimeProcessWiring(t *testing.T) {
	reader := NewPropertyConfigReader(configTestEnv{dir: fixtureDir(t, "separate")}, nil)
	process := &FatimaRuntimeProcess{builder: configTestBuilder{config: reader}}
	var runtime fatima.FatimaRuntime = process
	structured, ok := runtime.GetConfig().(fatima.StructuredConfig)
	require.True(t, ok)
	checkNode(t, structured, 18080, "server", "port")
	var apis []struct{ Name string }
	require.NoError(t, structured.DecodeNode(&apis, "services", "group", "api"))
	require.Len(t, apis, 1)
	assert.Equal(t, "qa-order-api", apis[0].Name)
	list, err := runtime.GetConfig().GetList("services.exclude")
	require.NoError(t, err)
	assert.Empty(t, list)
}

// These tests capture the synchronous stdout logger and must not run in parallel.
func TestStructuredLogsDoNotExposeValues(t *testing.T) {
	capture, err := os.CreateTemp(t.TempDir(), "log")
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = capture
	level := log.GetLevel()
	log.SetLevel(log.LOG_TRACE)
	defer func() { os.Stdout = old; log.SetLevel(level); capture.Close() }()
	dir := t.TempDir()
	writeTestFile(t, dir, "application.yaml", `secret.secret: b64:cHJpdmF0ZS1maXh0dXJlLXZhbHVl
---
items: [{token.secret: "b64:private-invalid!"}]
---
fatima: {profile: qa}
secret.secret: "private-fixture-value:encoded"
---
fatima: {profile: qa}
secret.secret: b64:cHJpdmF0ZS1maXh0dXJlLXZhbHVl
`)
	LoadApplicationConfig(dir, "qa", nil)
	writeTestFile(t, dir, "application.yaml", "secret: [private-fixture-value\n")
	LoadApplicationConfig(dir, "qa", nil)
	require.NoError(t, capture.Sync())
	data, err := os.ReadFile(capture.Name())
	require.NoError(t, err)
	output := string(data)
	assert.Contains(t, output, "multiple base documents")
	assert.Contains(t, output, "duplicate profile")
	assert.Contains(t, output, "overridden by")
	assert.Contains(t, output, "cannot load base config")
	assert.NotContains(t, output, "yaml array of complex types unsupported")
	for _, secret := range []string{"private-fixture-value", "cHJpdmF0ZS1maXh0dXJlLXZhbHVl", "private-invalid!"} {
		assert.NotContains(t, output, secret)
	}
}

func TestStructuredTypedContainers(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "application.yaml", `records: [{name: first, active: true}]
map: {one: 12}
secret.secret: b64:Zml4dHVyZS10b2tlbg==
`)
	loaded := LoadApplicationConfig(dir, "", nil)
	type record struct {
		Name   string
		Active bool
	}
	var records [1]*record
	require.NoError(t, loaded.Structured.DecodeNode(&records, "records"))
	assert.Equal(t, &record{Name: "first", Active: true}, records[0])
	var m map[string]uint16
	require.NoError(t, loaded.Structured.DecodeNode(&m, "map"))
	assert.Equal(t, map[string]uint16{"one": 12}, m)
	records[0].Name = "changed"
	m["one"] = 13
	checkNode(t, loaded.Structured, 12, "map", "one")
	loaded.Values["map.one"] = "changed"
	checkNode(t, loaded.Structured, 12, "map", "one")
	var unsupported struct {
		Name string `yaml:",inline"`
	}
	require.ErrorIs(t, loaded.Structured.DecodeNode(&unsupported, "map"), fatima.ErrInvalidConfigTarget)
	reader := newReaderFromDir(t, dir, "")
	_, err := reader.GetInt("secret.secret")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "fixture-token")
}
