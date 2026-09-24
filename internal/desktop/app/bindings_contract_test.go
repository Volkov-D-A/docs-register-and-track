package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/config"

	"github.com/stretchr/testify/require"
)

// The snapshot is the exact approved user-facing API, with no migration exceptions.
func TestWailsAPIContract(t *testing.T) {
	data, err := os.ReadFile("testdata/wails-api.json")
	require.NoError(t, err)
	var expected map[string][]string
	require.NoError(t, json.Unmarshal(data, &expected))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	releaseNotes, err := os.ReadFile("../../shared/releaseassets/current_release.yaml")
	require.NoError(t, err)
	appOptions, failure := NewWailsOptions(&config.Config{}, WailsOptionsParams{ReleaseNotesSource: releaseNotes})
	require.Nil(t, failure)
	t.Cleanup(func() { appOptions.OnShutdown(context.Background()) })
	actual := map[string][]string{}
	for _, binding := range appOptions.Bind {
		typ := reflect.TypeOf(binding)
		name := typ.Elem().Name()
		require.Equal(t, "github.com/Volkov-D-A/docs-register-and-track/internal/desktop/services", typ.Elem().PkgPath())
		require.NotContains(t, actual, name, "duplicate binding")
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i)
			require.NotEqual(t, "Startup", method.Name)
			require.NotEqual(t, "Shutdown", method.Name)
			require.NotEqual(t, "LogAction", method.Name)
			if strings.HasPrefix(method.Name, "Set") {
				require.Equal(t, "ThemeService.SetTheme", name+"."+method.Name)
			}
			seen := map[reflect.Type]bool{}
			for arg := 1; arg < method.Type.NumIn(); arg++ {
				assertUIType(t, method.Type.In(arg), seen)
			}
			for arg := 0; arg < method.Type.NumOut(); arg++ {
				assertUIType(t, method.Type.Out(arg), seen)
			}
			actual[name] = append(actual[name], method.Name)
		}
		sort.Strings(actual[name])
	}
	require.Equal(t, expected, actual, "review the approved user-facing API")
	for _, extension := range []string{"js", "d.ts"} {
		generated := map[string][]string{}
		files, err := filepath.Glob("../../../frontend/wailsjs/go/*/*." + extension)
		require.NoError(t, err)
		pattern := regexp.MustCompile(`export function (\w+)\(`)
		for _, file := range files {
			content, err := os.ReadFile(file)
			require.NoError(t, err)
			name := strings.TrimSuffix(filepath.Base(file), "."+extension)
			for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
				generated[name] = append(generated[name], match[1])
			}
			sort.Strings(generated[name])
		}
		require.Equal(t, expected, generated, "generated %s API differs", extension)
	}
}

// Inspect nested public fields too: a DTO must not conceal server types or callbacks.
func assertUIType(t *testing.T, typ reflect.Type, seen map[reflect.Type]bool) {
	t.Helper()
	if seen[typ] {
		return
	}
	seen[typ] = true
	pkg := typ.PkgPath()
	if strings.HasPrefix(pkg, "github.com/Volkov-D-A/docs-register-and-track/internal/") {
		require.True(t, pkg == "github.com/Volkov-D-A/docs-register-and-track/internal/dto" || pkg == "github.com/Volkov-D-A/docs-register-and-track/internal/models" ||
			(pkg == "github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient" && typ.Name() == "SessionState"), "non-contract type exposed: %v", typ)
	}
	switch typ.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		t.Fatalf("internal API type exposed: %v", typ)
	case reflect.Pointer, reflect.Slice, reflect.Array:
		assertUIType(t, typ.Elem(), seen)
	case reflect.Map:
		assertUIType(t, typ.Key(), seen)
		assertUIType(t, typ.Elem(), seen)
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.IsExported() {
				assertUIType(t, field.Type, seen)
			}
		}
	}
}
