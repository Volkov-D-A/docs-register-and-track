package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This is a migration ceiling, not approval to keep internal methods in the UI.
// Remove entries as services move; do not regenerate it to accept new setters.
func TestWailsAPIContract(t *testing.T) {
	data, err := os.ReadFile("testdata/wails-api.json")
	require.NoError(t, err)
	var expected map[string][]string
	require.NoError(t, json.Unmarshal(data, &expected))
	actual := map[string][]string{}
	for _, binding := range NewBindingsWailsOptions().Bind {
		typ := reflect.TypeOf(binding)
		name := typ.Elem().Name()
		require.NotContains(t, actual, name, "duplicate binding")
		for i := 0; i < typ.NumMethod(); i++ {
			actual[name] = append(actual[name], typ.Method(i).Name)
		}
		sort.Strings(actual[name])
	}
	require.Equal(t, expected, actual, "review UI contract and shrink legacy entries when migrating")
	for _, extension := range []string{"js", "d.ts"} {
		generated := map[string][]string{}
		files, err := filepath.Glob("../../frontend/wailsjs/go/*/*." + extension)
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
