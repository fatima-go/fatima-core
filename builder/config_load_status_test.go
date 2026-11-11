package builder

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fatima "github.com/fatima-go/fatima-core/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadStatus(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		want       map[string]string
		baseLoaded bool
		failed     []string
		missing    bool
	}{
		{name: "no files", want: map[string]string{}, missing: true},
		{name: "empty base", files: map[string]string{"application.yaml": ""}, want: map[string]string{}, baseLoaded: true},
		{name: "optional profile absent", files: map[string]string{"application.yml": "key: base"}, want: map[string]string{"key": "base"}, baseLoaded: true},
		{name: "profile only", files: map[string]string{"application.qa.yaml": "key: profile"}, want: map[string]string{"key": "profile"}},
		{name: "invalid base", files: map[string]string{"application.yaml": "key: [private-fixture-value"}, want: map[string]string{}, failed: []string{"application.yaml"}},
		{name: "invalid profile retains base", files: map[string]string{"application.yaml": "key: base", "application.qa.yaml": "key: [private-fixture-value"}, want: map[string]string{"key": "base"}, baseLoaded: true, failed: []string{"application.qa.yaml"}},
		{name: "invalid base retains profile", files: map[string]string{"application.yaml": "key: [private-fixture-value", "application.qa.yaml": "key: profile"}, want: map[string]string{"key": "profile"}, failed: []string{"application.yaml"}},
		{name: "both files fail", files: map[string]string{"application.yaml": "key: [private-fixture-value", "application.qa.yaml": "key: [private-fixture-value"}, want: map[string]string{}, failed: []string{"application.yaml", "application.qa.yaml"}},
		{name: "multidoc ignores broken separate profile", files: map[string]string{"application.yaml": "key: base\n---\nfatima: {profile: qa}\nkey: profile\n", "application.qa.yaml": "key: [private-fixture-value"}, want: map[string]string{"key": "profile"}, baseLoaded: true},
		{name: "invalid unselected document rejects file", files: map[string]string{"application.yaml": "key: base\n---\nfatima: {profile: prod}\nkey: [private-fixture-value"}, want: map[string]string{}, failed: []string{"application.yaml"}},
		{name: "properties", files: map[string]string{"application.properties": "key=base", "application.qa.properties": "key=profile"}, want: map[string]string{"key": "profile"}, baseLoaded: true},
		{name: "properties scanner failure", files: map[string]string{"application.properties": "key=base\nlong=" + strings.Repeat("x", 70000)}, want: map[string]string{}, failed: []string{"application.properties"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for file, content := range tt.files {
				writeTestFile(t, dir, file, content)
			}
			loaded := LoadApplicationConfig(dir, "qa", nil)
			assert.Equal(t, tt.want, loaded.Values)
			status := loaded.Structured
			require.NotNil(t, status)
			assert.Equal(t, tt.baseLoaded, status.HasBaseConfig())
			err := status.LoadError()
			if tt.missing {
				require.ErrorIs(t, err, fatima.ErrConfigNotFound)
				return
			}
			if len(tt.failed) == 0 {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, fatima.ErrConfigLoadFailed)
			assert.NotContains(t, err.Error(), "private-fixture-value")
			assert.NotContains(t, err.Error(), dir)
			joined, ok := err.(interface{ Unwrap() []error })
			require.True(t, ok)
			require.Len(t, joined.Unwrap(), len(tt.failed))
			for i, item := range joined.Unwrap() {
				var diagnostic *fatima.ConfigLoadError
				require.ErrorAs(t, item, &diagnostic)
				assert.Equal(t, tt.failed[i], diagnostic.File)
			}
			// Diagnostic mutation must not alter future status queries.
			var diagnostic *fatima.ConfigLoadError
			require.ErrorAs(t, err, &diagnostic)
			diagnostic.File = "mutated"
			diagnostic.Err = errors.New("mutated")
			assert.ErrorIs(t, status.LoadError(), fatima.ErrConfigLoadFailed)
			assert.NotContains(t, status.LoadError().Error(), "mutated")
		})
	}
}

func TestConfigLoadStatusReadFailure(t *testing.T) {
	for _, ext := range []string{"yaml", "properties"} {
		t.Run(ext, func(t *testing.T) {
			dir := t.TempDir()
			// A directory is an existing candidate that cannot be read as a file.
			require.NoError(t, os.Mkdir(filepath.Join(dir, "application."+ext), 0700))
			loaded := LoadApplicationConfig(dir, "qa", nil)
			require.ErrorIs(t, loaded.Structured.LoadError(), fatima.ErrConfigLoadFailed)
			assert.False(t, loaded.Structured.HasBaseConfig())
			assert.Empty(t, loaded.Values)
		})
	}
}

func TestConfigLoadStatusRuntimeProcessWiring(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "application.yaml", "key: base")
	writeTestFile(t, dir, "application.qa.yaml", "key: [broken")
	reader := NewPropertyConfigReader(configTestEnv{dir: dir}, nil)
	process := &FatimaRuntimeProcess{builder: configTestBuilder{config: reader}}
	var runtime fatima.FatimaRuntime = process
	status, ok := runtime.GetConfig().(fatima.ConfigLoadStatus)
	require.True(t, ok)
	require.ErrorIs(t, status.LoadError(), fatima.ErrConfigLoadFailed)
	assert.True(t, status.HasBaseConfig())
	value, err := runtime.GetConfig().GetString("key")
	require.NoError(t, err)
	assert.Equal(t, "base", value)
}

func TestConfigLoadStatusWithoutFiles(t *testing.T) {
	var unavailable *ConfigSnapshot
	require.ErrorIs(t, unavailable.LoadError(), fatima.ErrConfigNotFound)
	assert.False(t, unavailable.HasBaseConfig())
	flat := NewStringConfigSnapshot(map[string]string{"key": "value"})
	require.NoError(t, flat.LoadError())
	assert.False(t, flat.HasBaseConfig())
}
