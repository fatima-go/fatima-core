package runtime

import (
	"os"
	"path/filepath"
	"testing"

	fatima "github.com/fatima-go/fatima-core/v2"
	"github.com/fatima-go/fatima-core/v2/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredDefaultBuilderAndMockRuntime(t *testing.T) {
	t.Setenv(fatima.ENV_FATIMA_HOME, t.TempDir())
	env := NewMockFatimaEnv("structured-test")
	env.SetProfile("qa")
	dir := env.GetFolderGuide().GetAppFolder()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "application.yaml"), []byte("items: [{name: base}]\nport: 8080\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "application.qa.yaml"), []byte("items: []\nport: 18080\n"), 0600))
	// Exercise the production constructor path used by the runtime bootstrap.
	productionBuilder := getRuntimeBuilder(env, fatima.PROCESS_TYPE_UI)
	reader, ok := productionBuilder.GetConfig().(*builder.PropertyConfigReader)
	require.True(t, ok)
	mock := NewMockFatimaRuntime().SetEnv(env).SetConfig(reader)
	previous := process
	SetFatimaRuntimeForTest(mock)
	t.Cleanup(func() { SetFatimaRuntimeForTest(previous) })
	c := GetFatimaRuntime().GetConfig()
	structured, ok := c.(fatima.StructuredConfig)
	require.True(t, ok)
	value, exists := structured.LookupNode("items")
	require.True(t, exists)
	assert.Equal(t, []any{}, value)
	var port int
	require.NoError(t, structured.DecodeNode(&port, "port"))
	assert.Equal(t, 18080, port)
	values, err := c.GetList("items")
	require.NoError(t, err)
	assert.Equal(t, []string{}, values)
}

func TestMockConfigStructuredResources(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n"), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "resources"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "resources", "application.yaml"), []byte("items: [{name: resource}]\n"), 0600))
	t.Chdir(dir)
	t.Setenv(fatima.ENV_FATIMA_PROFILE, "")
	c := NewMockConfig()
	var items []struct{ Name string }
	require.NoError(t, c.DecodeNode(&items, "items"))
	require.Len(t, items, 1)
	assert.Equal(t, "resource", items[0].Name)
	_, err := c.GetList("items")
	require.ErrorContains(t, err, "unsupported value type")
	flat := NewMockConfigWithMap(map[string]string{"a.b": "value"})
	value, exists := flat.LookupNode("a.b")
	require.True(t, exists)
	assert.Equal(t, "value", value)
}
