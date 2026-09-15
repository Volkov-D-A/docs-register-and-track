// buildmeta also acts as the Go compiler for Wails, refreshing metadata on every rebuild.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/buildinfo"
)

func git(root string, args ...string) (string, error) {
	c := exec.Command("git", append([]string{"-C", root}, args...)...)
	b, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}
func excluded(path string) bool {
	for _, prefix := range []string{"frontend/wailsjs/", "frontend/dist/", "frontend/node_modules/", "frontend/.test-build/", "build/bin/", "build/release-evidence/", "build/performance/", "build/transition-evidence/", "config/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return path == "docflow-res.syso" || path == ".env" || path == "server-config.json"
}
func snapshot(root string, local bool) (buildinfo.Identity, error) {
	shallow, err := git(root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return buildinfo.Identity{}, err
	}
	if shallow != "false" {
		return buildinfo.Identity{}, fmt.Errorf("full Git history required; fetch with --unshallow")
	}
	number, err := git(root, "rev-list", "--count", "HEAD")
	if err != nil {
		return buildinfo.Identity{}, err
	}
	revision, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		return buildinfo.Identity{}, err
	}
	changed, err := git(root, "diff", "--name-only", "-z", "HEAD")
	if err != nil {
		return buildinfo.Identity{}, err
	}
	untracked, err := git(root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return buildinfo.Identity{}, err
	}
	dirty := false
	for _, p := range strings.Split(changed+untracked, "\x00") {
		if p != "" && !excluded(p) {
			dirty = true
		}
	}
	identity := buildinfo.Identity{Number: number, Revision: revision}
	if !dirty {
		return identity, nil
	}
	if !local {
		return identity, fmt.Errorf("source tree has uncommitted changes; commit them or use a local development target")
	}
	tracked, err := git(root, "ls-files", "-z")
	if err != nil {
		return identity, err
	}
	paths := strings.Split(tracked+untracked, "\x00")
	sort.Strings(paths)
	hash := sha256.New()
	previous := ""
	for _, p := range paths {
		if p == "" || p == previous || excluded(p) {
			continue
		}
		previous = p
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return identity, err
		}
		if !bytes.ContainsRune(data, 0) {
			data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		}
		fmt.Fprintf(hash, "%d:%s:%d:", len(p), p, len(data))
		hash.Write(data)
	}
	identity.Fingerprint = fmt.Sprintf("%x", hash.Sum(nil))
	return identity, nil
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: buildmeta identity|ldflags|build [go arguments]")
	}
	if args[0] == "from-identity" {
		if len(args) != 2 {
			return fmt.Errorf("from-identity requires one identity")
		}
		identity, err := buildinfo.Parse(args[1])
		if err != nil {
			return err
		}
		if identity.Fingerprint != "" && os.Getenv("DOCFLOW_LOCAL_BUILD") != "1" {
			return fmt.Errorf("Docker distribution requires a clean source identity")
		}
		fmt.Print(identity.LDFlags())
		return nil
	}
	if args[0] == "identity" || args[0] == "ldflags" || args[0] == "build" {
		root, err := git(".", "rev-parse", "--show-toplevel")
		if err != nil {
			return err
		}
		identity, err := snapshot(root, os.Getenv("DOCFLOW_LOCAL_BUILD") == "1")
		if err != nil {
			return err
		}
		if args[0] == "identity" {
			fmt.Println(identity.String())
			return nil
		}
		if args[0] == "ldflags" {
			fmt.Println(identity.LDFlags())
			return nil
		}
		// Merge Wails' own linker flags rather than replacing them.
		args = append([]string(nil), args...)
		merged := false
		for n := 1; n < len(args); n++ {
			if args[n] == "-ldflags" && n+1 < len(args) {
				args[n+1] += " " + identity.LDFlags()
				merged = true
				break
			}
			if strings.HasPrefix(args[n], "-ldflags=") {
				args[n] += " " + identity.LDFlags()
				merged = true
				break
			}
		}
		if !merged {
			args = append([]string{"build", "-ldflags", identity.LDFlags()}, args[1:]...)
		}
	}
	cmd := exec.Command("go", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
