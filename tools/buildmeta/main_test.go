package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.test"}, {"config", "user.name", "Test"}} {
		if _, err := git(root, args...); err != nil {
			t.Fatal(err)
		}
	}
	write(t, root, "source.go", "package example\n")
	if _, err := git(root, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := git(root, "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	return root
}
func write(t *testing.T, root, name, data string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}
func TestSnapshot(t *testing.T) {
	root := fixture(t)
	a, err := snapshot(root, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := snapshot(root, false)
	if err != nil || !a.Matches(b) {
		t.Fatal(b, err)
	}
	write(t, root, "source.go", "package changed\n")
	if _, err := snapshot(root, false); err == nil {
		t.Fatal("dirty distribution accepted")
	}
	b, err = snapshot(root, true)
	if err != nil || b.Fingerprint == "" || a.Matches(b) {
		t.Fatal(b, err)
	}
	write(t, root, "source.go", "package changed\r\n")
	c, err := snapshot(root, true)
	if err != nil || !b.Matches(c) {
		t.Fatal("line endings changed identity", c, err)
	}
	write(t, root, "frontend/wailsjs/generated.go", "generated")
	c, err = snapshot(root, true)
	if err != nil || !b.Matches(c) {
		t.Fatal("bindings changed identity", c, err)
	}
	write(t, root, "new.go", "package newfile\n")
	c, err = snapshot(root, true)
	if err != nil || b.Matches(c) {
		t.Fatal("untracked source ignored", c, err)
	}
	if err := os.Remove(filepath.Join(root, "source.go")); err != nil {
		t.Fatal(err)
	}
	d, err := snapshot(root, true)
	if err != nil || c.Matches(d) {
		t.Fatal("deletion ignored", d, err)
	}
}
func TestSnapshotRejectsMissingAndShallowGit(t *testing.T) {
	if _, err := snapshot(t.TempDir(), true); err == nil {
		t.Fatal("missing git accepted")
	}
	root := fixture(t)
	clone := filepath.Join(t.TempDir(), "clone")
	if out, err := exec.Command("git", "clone", "--depth=1", "file://"+root, clone).CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	if _, err := snapshot(clone, true); err == nil {
		t.Fatal("shallow git accepted")
	}
}
