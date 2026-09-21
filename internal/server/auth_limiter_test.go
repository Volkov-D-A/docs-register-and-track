package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuthenticationIPBudgetCountsPendingAndSuccessfulAttempts(t *testing.T) {
	for _, successful := range []bool{false, true} {
		t.Run(fmt.Sprintf("successful=%t", successful), func(t *testing.T) {
			api := &managementAPI{}
			now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
			for i := 0; i < maxAuthenticationAttemptsPerIPPerMinute; i++ {
				key := fmt.Sprintf("192.0.2.1\x00user-%d", i)
				require.True(t, api.authenticationAllowed(key, now))
				if successful {
					api.clearAuthenticationFailures(key)
				}
			}
			require.False(t, api.authenticationAllowed("192.0.2.1\x00new-login", now))
			require.True(t, api.authenticationAllowed("192.0.2.2\x00new-login", now))
			require.True(t, api.authenticationAllowed("192.0.2.1\x00new-login", now.Add(time.Minute)))
		})
	}
}

func TestAuthenticationGlobalBudgetIsAtomicForConcurrentAttempts(t *testing.T) {
	api := &managementAPI{}
	now := time.Now()
	var admitted atomic.Int32
	var group sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 2*maxAuthenticationAttemptsPerMinute; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			<-start
			// Each attempt has a different IPv6 source, bypassing only the IP budget.
			if api.authenticationAllowed(fmt.Sprintf("2001:db8::%x\x00user", i+1), now) {
				admitted.Add(1)
			}
		}(i)
	}
	close(start)
	group.Wait()
	require.Equal(t, int32(maxAuthenticationAttemptsPerMinute), admitted.Load())
	require.True(t, api.authenticationAllowed("192.0.2.1\x00user", now.Add(time.Minute)))
}

func TestAuthenticationRetainsLoginFailureLimit(t *testing.T) {
	api := &managementAPI{}
	now := time.Now()
	key := "192.0.2.1\x00user"
	for i := 0; i < 5; i++ {
		require.True(t, api.authenticationAllowed(key, now))
		api.recordAuthenticationFailure(key, now)
	}
	require.False(t, api.authenticationAllowed(key, now))
	require.True(t, api.authenticationAllowed("192.0.2.1\x00other-user", now))
	require.True(t, api.authenticationAllowed(key, now.Add(time.Minute)))
}

func TestAuthenticationRejectsNewKeysWhenTablesAreFull(t *testing.T) {
	for _, table := range []string{"failures", "ips"} {
		t.Run(table, func(t *testing.T) {
			api := &managementAPI{}
			now := time.Now()
			entries := make(map[string]authFailure)
			for i := 0; i < maxAuthenticationFailureKeys; i++ {
				entries[fmt.Sprintf("existing-%d", i)] = authFailure{count: 1, resetAt: now.Add(time.Minute)}
			}
			if table == "failures" {
				api.authFailures = entries
			} else {
				api.authIPAttempts = entries
			}
			for i := 0; i < 10; i++ {
				require.False(t, api.authenticationAllowed("192.0.2.1\x00new-user", now))
			}
			require.Len(t, entries, maxAuthenticationFailureKeys)
			require.True(t, api.authenticationAllowed("192.0.2.1\x00new-user", now.Add(time.Minute)))
			require.LessOrEqual(t, len(entries), 1)
		})
	}
}

func TestAuthenticationRoutesShareBudgetBeforeUserLookup(t *testing.T) {
	api := &managementAPI{}
	// No user stores are configured: reaching a lookup would panic. Reserve the
	// source budget without completing requests to model pending password checks.
	now := time.Now()
	for i := 0; i < maxAuthenticationAttemptsPerIPPerMinute; i++ {
		require.True(t, api.authenticationAllowed(fmt.Sprintf("192.0.2.1\x00user-%d", i), now))
	}
	for _, route := range []struct {
		name, body string
		handler    http.HandlerFunc
	}{
		{"login", `{"login":"new-user","password":"Passw0rd!"}`, api.login},
		{"required password", `{"login":"new-user","oldPassword":"Passw0rd!","newPassword":"Another1!"}`, api.changeRequiredPassword},
		{"migration auth", "", func(w http.ResponseWriter, r *http.Request) { api.authenticateAdmin(w, r) }},
	} {
		t.Run(route.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(route.body))
			req.RemoteAddr = "192.0.2.1:12345"
			req.SetBasicAuth("new-user", "Passw0rd!")
			response := httptest.NewRecorder()
			route.handler(response, req)
			require.Equal(t, http.StatusTooManyRequests, response.Code)
			require.Contains(t, response.Body.String(), "authentication_rate_limited")
		})
	}
}
