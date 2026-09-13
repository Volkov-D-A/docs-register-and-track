package architecture

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/Volkov-D-A/docs-register-and-track/"

func within(path, prefix string) bool { return path == prefix || strings.HasPrefix(path, prefix+"/") }

// Check transitive production dependencies, including both executable roots.
func forbidden(owner, dependency string) bool {
	for _, retired := range []string{"services", "app", "serverclient", "config", "database", "repository", "storage", "backup", "liveevents", "outbox", "coordination", "security"} {
		if within(dependency, module+"internal/"+retired) {
			return true
		}
	}
	desktop := owner == strings.TrimSuffix(module, "/") || within(owner, module+"internal/desktop")
	server := within(owner, module+"internal/server") || within(owner, module+"cmd/docflow-server")
	testSupport := within(owner, module+"internal/testutil") || within(owner, module+"internal/mocks") || within(owner, module+"internal/architecture")
	shared := within(owner, module+"internal") && !desktop && !server && !testSupport
	if (server || shared) && (within(dependency, module+"internal/desktop") || within(dependency, "github.com/wailsapp/wails/v2")) {
		return true
	}
	if (desktop || shared) && within(dependency, module+"internal/server") {
		return true
	}
	return false
}

func TestProductionImportBoundaries(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-json", "./...")
	cmd.Dir = "../.."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for {
		var pkg struct {
			ImportPath string
			Deps       []string
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		for _, dependency := range pkg.Deps {
			if forbidden(pkg.ImportPath, dependency) {
				t.Errorf("%s transitively imports forbidden %s", pkg.ImportPath, dependency)
			}
		}
	}
}

func TestBoundaryPolicy(t *testing.T) {
	for _, tc := range []struct {
		owner, dependency string
		blocked           bool
	}{
		{"internal/logger", "internal/desktop/serverclient", true},
		{"internal/logger", "internal/server/config", true},
		{"internal/server/logging", "internal/desktop/logging", true},
		{"internal/desktop/config", "internal/server/config", true},
		{"internal/server/logging", "internal/logger", false},
		{"internal/desktop/app", "internal/background", false},
		{"internal/background", "internal/server/config", true},
		{"internal/background", "internal/desktop/serverclient", true},
		{"internal/desktop/services", "internal/services", true},
		{"internal/desktop/services", "internal/server/services", true},
		{"internal/server/services", "internal/desktop/serverclient", true},
		{"internal/server/ports", "internal/services", true},
		{"internal/operations", "internal/desktop/services", true},
		{"internal/dto", "internal/server/database", true},
		{"internal/desktop/serverclient", "internal/server/database", true},
		{"internal/background", "internal/server/database", true},
		{"internal/desktop/services", "internal/server/backup", true},
		{"internal/desktop/services", "internal/server/backup/smb", true},
		{"internal/models", "internal/server/liveevents", true},
		{"internal/server/backup", "internal/desktop/serverclient", true},
		{"internal/server/backup/smb", "internal/desktop/services", true},
		{"internal/server/liveevents", "internal/services", true},
		{"internal/server/liveevents", "internal/app", true},
		{"internal/server/backup", "internal/server/storage", false},
		{"internal/server/backup", "internal/server/database", false},
		{"internal/server/backup", "internal/server/liveevents", false},
		{"internal/desktop/services", "internal/dto", false},
		{"internal/server/services", "internal/server/ports", false},
		{"internal/server", "internal/desktop/serverclient", true},
		{"cmd/docflow-server", "internal/desktop/app", true},
		{"", "internal/server/storage", true},
		{"internal/startupdiag", "internal/server/config", true},
		{"internal/future_shared", "internal/desktop/config", true},
	} {
		if got := forbidden(strings.TrimSuffix(module+tc.owner, "/"), module+tc.dependency); got != tc.blocked {
			t.Errorf("%s -> %s: got %v", tc.owner, tc.dependency, got)
		}
	}
	if !forbidden(module+"internal/server/services", "github.com/wailsapp/wails/v2/pkg/runtime") {
		t.Fatal("server must reject Wails")
	}
	for _, owner := range []string{"server/backup", "server/backup/smb", "server/liveevents"} {
		if !forbidden(module+"internal/"+owner, "github.com/wailsapp/wails/v2/pkg/runtime") {
			t.Errorf("%s must reject Wails", owner)
		}
	}
}

func TestRetiredPackageDirectoriesStayRemoved(t *testing.T) {
	for _, name := range []string{"services", "app", "serverclient", "config", "database", "repository", "storage", "backup", "liveevents", "outbox", "coordination", "security"} {
		_, err := os.Stat(filepath.Join("..", name))
		if !os.IsNotExist(err) {
			t.Errorf("retired directory internal/%s must stay removed (stat: %v)", name, err)
		}
	}
}
