// Package buildinfo identifies source snapshots independently of public releases.
package buildinfo

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Values are injected by tools/buildmeta. Empty values deliberately fail closed.
var Number, Revision, Fingerprint string

const Protocol = 2

type Identity struct {
	Number      string `json:"number"`
	Revision    string `json:"revision"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

func Current() Identity { return Identity{Number, Revision, Fingerprint} }

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
var fingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (i Identity) Valid() bool {
	n, err := strconv.ParseUint(i.Number, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == i.Number && revisionPattern.MatchString(i.Revision) && (i.Fingerprint == "" || fingerprintPattern.MatchString(i.Fingerprint))
}
func (i Identity) Matches(other Identity) bool { return i.Valid() && other.Valid() && i == other }
func (i Identity) Version(release string) string {
	if !i.Valid() {
		return ""
	}
	return release + "." + i.Number
}
func (i Identity) String() string { return i.Number + ":" + i.Revision + ":" + i.Fingerprint }
func Parse(raw string) (Identity, error) {
	p := strings.Split(raw, ":")
	if len(p) != 3 {
		return Identity{}, fmt.Errorf("invalid build identity")
	}
	i := Identity{p[0], p[1], p[2]}
	if !i.Valid() {
		return Identity{}, fmt.Errorf("invalid build identity")
	}
	return i, nil
}
func (i Identity) LDFlags() string {
	const pkg = "github.com/Volkov-D-A/docs-register-and-track/internal/buildinfo."
	return "-X " + pkg + "Number=" + i.Number + " -X " + pkg + "Revision=" + i.Revision + " -X " + pkg + "Fingerprint=" + i.Fingerprint
}
