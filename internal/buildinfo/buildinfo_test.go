package buildinfo

import (
	"strings"
	"testing"
)

func TestIdentityFailsClosed(t *testing.T) {
	a := Identity{"213", strings.Repeat("a", 40), ""}
	if !a.Matches(a) || a.Version("1.0.7") != "1.0.7.213" {
		t.Fatal(a)
	}
	for _, b := range []Identity{{}, {"0213", a.Revision, ""}, {"214", a.Revision, ""}, {a.Number, strings.Repeat("b", 40), ""}, {a.Number, a.Revision, strings.Repeat("a", 64)}} {
		if a.Matches(b) {
			t.Fatalf("unexpected match: %+v", b)
		}
	}
	if (Identity{}).Matches(Identity{}) {
		t.Fatal("unknown builds must not match")
	}
	for _, raw := range []string{"", "0:" + a.Revision + ":", "1:bad:", "1:" + a.Revision + ":bad"} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}
