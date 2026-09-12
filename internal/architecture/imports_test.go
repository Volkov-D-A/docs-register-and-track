package architecture

import (
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

const module = "github.com/Volkov-D-A/docs-register-and-track/"

func within(path, prefix string) bool { return path == prefix || strings.HasPrefix(path, prefix+"/") }

// Existing mixed roots are intentionally not exempted as dependencies of new
// packages. Every new descendant is enrolled automatically from its first file.
func forbidden(owner, dependency string) bool {
	// Step 2: status consumers must remain independent of migration infrastructure.
	if (within(owner, module+"internal/serverclient") || within(owner, module+"internal/background")) && within(dependency, module+"internal/database") {
		return true
	}
	desktop := within(owner, module+"internal/desktop")
	server := within(owner, module+"internal/server") && owner != module+"internal/server"
	// Server infrastructure added after step 17 is protected before its move.
	server = server || within(owner, module+"internal/backup") || within(owner, module+"internal/liveevents")
	shared := false
	for _, root := range []string{"dto", "models", "operations", "observability", "releaseassets", "attachmentname"} {
		shared = shared || within(owner, module+"internal/"+root)
	}
	if !desktop && !server && !shared {
		return false
	}
	if within(dependency, module+"internal/services") || within(dependency, module+"internal/app") {
		return true
	}
	if (server || shared) && (within(dependency, module+"internal/desktop") || within(dependency, module+"internal/serverclient") || within(dependency, "github.com/wailsapp/wails/v2")) {
		return true
	}
	if desktop || shared {
		for _, root := range []string{"server", "database", "repository", "storage", "outbox", "background", "coordination", "backup", "liveevents"} {
			if within(dependency, module+"internal/"+root) {
				return true
			}
		}
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
		{"internal/desktop/services", "internal/services", true},
		{"internal/desktop/services", "internal/server/services", true},
		{"internal/server/services", "internal/serverclient", true},
		{"internal/server/ports", "internal/services", true},
		{"internal/operations", "internal/desktop/services", true},
		{"internal/dto", "internal/database", true},
		{"internal/serverclient", "internal/database", true},
		{"internal/background", "internal/database", true},
		{"internal/desktop/services", "internal/backup", true},
		{"internal/desktop/services", "internal/backup/smb", true},
		{"internal/models", "internal/liveevents", true},
		{"internal/backup", "internal/serverclient", true},
		{"internal/backup/smb", "internal/desktop/services", true},
		{"internal/liveevents", "internal/services", true},
		{"internal/liveevents", "internal/app", true},
		{"internal/backup", "internal/storage", false},
		{"internal/backup", "internal/database", false},
		{"internal/backup", "internal/liveevents", false},
		{"internal/desktop/services", "internal/dto", false},
		{"internal/server/services", "internal/server/ports", false},
		{"internal/services", "internal/database", false},
	} {
		if got := forbidden(module+tc.owner, module+tc.dependency); got != tc.blocked {
			t.Errorf("%s -> %s: got %v", tc.owner, tc.dependency, got)
		}
	}
	if !forbidden(module+"internal/server/services", "github.com/wailsapp/wails/v2/pkg/runtime") {
		t.Fatal("server must reject Wails")
	}
	for _, owner := range []string{"backup", "backup/smb", "liveevents"} {
		if !forbidden(module+"internal/"+owner, "github.com/wailsapp/wails/v2/pkg/runtime") {
			t.Errorf("%s must reject Wails", owner)
		}
	}
}
