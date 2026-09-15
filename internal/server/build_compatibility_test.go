package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/buildinfo"
	"github.com/stretchr/testify/assert"
)

func TestBuildCompatibility(t *testing.T) {
	api, _, _, _ := testManagementAPI(t)
	a := buildinfo.Identity{Number: "213", Revision: strings.Repeat("a", 40)}
	api.serverBuild = a
	for _, tc := range []struct {
		name, identity, protocol, code string
		status                         int
	}{
		{"matching", a.String(), "2", "compatible", 200},
		{"legacy", "", "", "build_identity_required", 200},
		{"protocol", a.String(), "1", "build_identity_required", 200},
		{"number", "214:" + a.Revision + ":", "2", "build_mismatch", 200},
		{"branch", "213:" + strings.Repeat("b", 40) + ":", "2", "build_mismatch", 200},
		{"dirty", a.String() + strings.Repeat("c", 64), "2", "build_mismatch", 200},
		{"invalid", "bad", "2", "invalid_client_build", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := url.Values{"clientVersion": {"1.0.6"}, "clientBuild": {tc.identity}, "buildProtocol": {tc.protocol}}
			res := httptest.NewRecorder()
			api.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/system/compatibility?"+query.Encode(), nil))
			assert.Equal(t, tc.status, res.Code)
			assert.Contains(t, res.Body.String(), `"code":"`+tc.code+`"`)
		})
	}
}
