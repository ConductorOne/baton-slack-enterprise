package connector

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	enterprise "github.com/conductorone/baton-slack-enterprise/pkg/connector/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type recordedRequest struct {
	method string
	path   string
	form   url.Values
	body   []byte
}

type guardServer struct {
	srv      *httptest.Server
	mu       sync.Mutex
	requests []recordedRequest
	// cannedError overrides the default {"ok":true} response body when set.
	cannedError []byte
}

func newGuardServer(t *testing.T) *guardServer {
	t.Helper()

	g := &guardServer{}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		body, _ := io.ReadAll(r.Body)
		g.mu.Lock()
		g.requests = append(g.requests, recordedRequest{
			method: r.Method,
			path:   r.URL.Path,
			form:   r.PostForm,
			body:   body,
		})
		canned := g.cannedError
		g.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/users.info" {
			_, _ = w.Write([]byte(`{"ok":true,"user":{"profile":{"team":"T123"}}}`))
			return
		}
		if canned != nil {
			_, _ = w.Write(canned)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(g.srv.Close)

	return g
}

func (g *guardServer) count(t *testing.T) int {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.requests)
}

func (g *guardServer) paths(t *testing.T) []string {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	for _, r := range g.requests {
		out = append(out, r.path)
	}
	return out
}

func (g *guardServer) lastForm(t *testing.T) url.Values {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	require.NotEmpty(t, g.requests)
	return g.requests[len(g.requests)-1].form
}

// lastJSON decodes the last recorded request body (for postJSON endpoints).
func (g *guardServer) lastJSON(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	require.NotEmpty(t, g.requests)
	var out map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(g.requests[len(g.requests)-1].body, &out))
	return out
}

// jsonArrayFirst returns the first element of a JSON array field.
func jsonArrayFirst(t *testing.T, m map[string]json.RawMessage, key string) string {
	t.Helper()
	var arr []string
	require.NoError(t, json.Unmarshal(m[key], &arr))
	require.NotEmpty(t, arr)
	return arr[0]
}

// newGuardBuilder builds an enterpriseRoleBuilder against the guard server
// with the given orgID. The server serves /api/users.info so getTeamIDForUser
// succeeds for the granular fall-through tests.
func newGuardBuilder(t *testing.T, g *guardServer, orgID string) *enterpriseRoleType {
	t.Helper()

	// The connector package's client is constructed via enterprise.NewClient.
	c, err := enterprise.NewClient(g.srv.Client(), "token", "bot-token", orgID, false, g.srv.URL)
	require.NoError(t, err)

	return enterpriseRoleBuilder(orgID, c)
}

func guardGrantInput(roleID string) (*v2.Resource, *v2.Entitlement) {
	principal := &v2.Resource{Id: &v2.ResourceId{ResourceType: resourceTypeUser.Id, Resource: "U1"}}
	ent := &v2.Entitlement{Resource: &v2.Resource{Id: &v2.ResourceId{ResourceType: resourceTypeEnterpriseRole.Id, Resource: roleID}}}
	return principal, ent
}

func guardRevokeInput(roleID string) *v2.Grant {
	principal := &v2.Resource{Id: &v2.ResourceId{ResourceType: resourceTypeUser.Id, Resource: "U1"}}
	ent := &v2.Entitlement{Resource: &v2.Resource{Id: &v2.ResourceId{ResourceType: resourceTypeEnterpriseRole.Id, Resource: roleID}}}
	return &v2.Grant{Entitlement: ent, Principal: principal}
}

const guardTestOrgID = "E123"

// T1/T2: org-role grants route to admin.users.setAdmin/setOwner with the org
// ID as team_id — never admin.roles.addAssignments (AC 1, 2, 6).
func TestEnterpriseRoleGrant_OrgRolesUseOrgLevelAPI(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		roleID   string
		wantPath string
	}{
		{OrganizationAdminID, "/api/admin.users.setAdmin"},
		{OrganizationOwnerID, "/api/admin.users.setOwner"},
	} {
		t.Run(tc.roleID, func(t *testing.T) {
			g := newGuardServer(t)
			b := newGuardBuilder(t, g, guardTestOrgID)

			principal, ent := guardGrantInput(tc.roleID)
			_, err := b.Grant(ctx, principal, ent)
			require.NoError(t, err)

			paths := g.paths(t)
			require.Contains(t, paths, tc.wantPath)
			require.NotContains(t, paths, "/api/admin.roles.addAssignments")
			form := g.lastForm(t)
			require.Equal(t, guardTestOrgID, form.Get("team_id"))
			require.Equal(t, "U1", form.Get("user_id"))
		})
	}
}

// T3: revoking both org roles downgrades via admin.users.setRegular with the
// org ID — never admin.roles.removeAssignments (AC 3, 6).
func TestEnterpriseRoleRevoke_OrgRolesDowngradeToRegular(t *testing.T) {
	ctx := context.Background()
	for _, roleID := range []string{OrganizationAdminID, OrganizationOwnerID} {
		t.Run(roleID, func(t *testing.T) {
			g := newGuardServer(t)
			b := newGuardBuilder(t, g, guardTestOrgID)

			_, err := b.Revoke(ctx, guardRevokeInput(roleID))
			require.NoError(t, err)

			paths := g.paths(t)
			require.Contains(t, paths, "/api/admin.users.setRegular")
			require.NotContains(t, paths, "/api/admin.roles.removeAssignments")
			form := g.lastForm(t)
			require.Equal(t, guardTestOrgID, form.Get("team_id"))
			require.Equal(t, "U1", form.Get("user_id"))
		})
	}
}

// T4: organization_primary_owner grant and revoke are refused before any HTTP
// call, with a PermissionDenied error carrying the baton prefix (AC 4, 7).
func TestEnterpriseRole_PrimaryOwnerRefusedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	for _, op := range []struct {
		name  string
		grant bool
	}{
		{"grant", true},
		{"revoke", false},
	} {
		t.Run(op.name, func(t *testing.T) {
			g := newGuardServer(t)
			b := newGuardBuilder(t, g, guardTestOrgID)

			var err error
			if op.grant {
				principal, ent := guardGrantInput(OrganizationPrimaryOwnerID)
				_, err = b.Grant(ctx, principal, ent)
			} else {
				_, err = b.Revoke(ctx, guardRevokeInput(OrganizationPrimaryOwnerID))
			}

			require.Error(t, err)
			require.Equal(t, codes.PermissionDenied, status.Code(err))
			require.Contains(t, err.Error(), "baton-slack-enterprise:")
			require.Zero(t, g.count(t))
		})
	}
}

// T5: granular system roles keep falling through to
// admin.roles.addAssignments with the role ID and the user's team (AC 5, 6).
func TestEnterpriseRoleGrant_GranularRolesUnchanged(t *testing.T) {
	ctx := context.Background()
	g := newGuardServer(t)
	b := newGuardBuilder(t, g, guardTestOrgID)

	principal, ent := guardGrantInput(UserAdmin)
	_, err := b.Grant(ctx, principal, ent)
	require.NoError(t, err)

	paths := g.paths(t)
	require.Contains(t, paths, "/api/users.info")
	require.Contains(t, paths, "/api/admin.roles.addAssignments")
	require.NotContains(t, paths, "/api/admin.users.setAdmin")
	js := g.lastJSON(t)
	require.Equal(t, `"`+UserAdmin+`"`, string(js["role_id"]))
	require.Equal(t, "T123", jsonArrayFirst(t, js, "entity_ids"))
}

// T6: idempotent grants — the client wraps Slack errors as
// "Slack API error: <code>"; matching grants GrantAlreadyExists (AC 7).
func TestEnterpriseRoleGrant_IdempotentAlreadyExists(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		roleID   string
		slackErr string
	}{
		{OrganizationAdminID, "already_admin"},
		{OrganizationOwnerID, "already_owner"},
	} {
		t.Run(tc.roleID, func(t *testing.T) {
			g := newGuardServer(t)
			g.mu.Lock()
			g.cannedError = []byte(`{"ok":false,"error":"` + tc.slackErr + `"}`)
			g.mu.Unlock()
			b := newGuardBuilder(t, g, guardTestOrgID)

			principal, ent := guardGrantInput(tc.roleID)
			ann, err := b.Grant(ctx, principal, ent)
			require.NoError(t, err)
			require.True(t, ann.Contains(&v2.GrantAlreadyExists{}))
		})
	}
}

// T7: idempotent revokes — already_regular and invalid_role_for_user both
// yield GrantAlreadyRevoked (AC 7).
func TestEnterpriseRoleRevoke_IdempotentAlreadyRevoked(t *testing.T) {
	ctx := context.Background()
	for _, slackErr := range []string{"already_regular", "invalid_role_for_user"} {
		t.Run(slackErr, func(t *testing.T) {
			g := newGuardServer(t)
			g.mu.Lock()
			g.cannedError = []byte(`{"ok":false,"error":"` + slackErr + `"}`)
			g.mu.Unlock()
			b := newGuardBuilder(t, g, guardTestOrgID)

			ann, err := b.Revoke(ctx, guardRevokeInput(OrganizationAdminID))
			require.NoError(t, err)
			require.True(t, ann.Contains(&v2.GrantAlreadyRevoked{}))
		})
	}
}

// T8: an empty enterprise ID is rejected inside the org arms before any HTTP
// call with InvalidArgument — including for organization_primary_owner, whose
// guard order is locked (guard first) (I5).
func TestEnterpriseRole_EmptyEnterpriseIDGuard(t *testing.T) {
	ctx := context.Background()
	for _, op := range []struct {
		name   string
		roleID string
		grant  bool
	}{
		{"grant admin", OrganizationAdminID, true},
		{"grant primary owner", OrganizationPrimaryOwnerID, true},
		{"revoke admin", OrganizationAdminID, false},
		{"revoke primary owner", OrganizationPrimaryOwnerID, false},
	} {
		t.Run(op.name, func(t *testing.T) {
			g := newGuardServer(t)
			b := newGuardBuilder(t, g, "")

			var err error
			if op.grant {
				principal, ent := guardGrantInput(op.roleID)
				_, err = b.Grant(ctx, principal, ent)
			} else {
				_, err = b.Revoke(ctx, guardRevokeInput(op.roleID))
			}

			require.Error(t, err)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.Zero(t, g.count(t))
		})
	}
}

// Unmatched org-path Slack errors fail closed: wrapped error, no annotation,
// no fall-through to the granular path (I7).
func TestEnterpriseRoleGrant_UnmatchedErrorFailsClosed(t *testing.T) {
	ctx := context.Background()
	g := newGuardServer(t)
	g.mu.Lock()
	g.cannedError = []byte(`{"ok":false,"error":"user_not_found"}`)
	g.mu.Unlock()
	b := newGuardBuilder(t, g, guardTestOrgID)

	principal, ent := guardGrantInput(OrganizationAdminID)
	ann, err := b.Grant(ctx, principal, ent)
	require.Error(t, err)
	require.False(t, ann.Contains(&v2.GrantAlreadyExists{}))
	require.NotContains(t, g.paths(t), "/api/admin.roles.addAssignments")
}

// T9 tripwire: every organizationRoles key must route to the org branch with
// the correct org-level endpoint and org ID; every systemRoles key must fall
// through to admin.roles.addAssignments with its exact role ID.
func TestEnterpriseRole_RoutingTripwire(t *testing.T) {
	ctx := context.Background()

	t.Run("org roles", func(t *testing.T) {
		for roleID := range organizationRoles {
			t.Run(roleID, func(t *testing.T) {
				g := newGuardServer(t)
				b := newGuardBuilder(t, g, guardTestOrgID)

				principal, ent := guardGrantInput(roleID)
				_, err := b.Grant(ctx, principal, ent)
				if roleID == OrganizationPrimaryOwnerID {
					require.Error(t, err)
					require.Zero(t, g.count(t))
					return
				}

				require.NoError(t, err)
				wantPath := "/api/admin.users.setAdmin"
				if roleID == OrganizationOwnerID {
					wantPath = "/api/admin.users.setOwner"
				}
				paths := g.paths(t)
				require.Contains(t, paths, wantPath)
				require.Equal(t, guardTestOrgID, g.lastForm(t).Get("team_id"))
			})
		}
	})

	t.Run("system roles", func(t *testing.T) {
		for roleID := range systemRoles {
			t.Run(roleID, func(t *testing.T) {
				g := newGuardServer(t)
				b := newGuardBuilder(t, g, guardTestOrgID)

				principal, ent := guardGrantInput(roleID)
				_, err := b.Grant(ctx, principal, ent)
				require.NoError(t, err)

				paths := g.paths(t)
				require.Contains(t, paths, "/api/admin.roles.addAssignments")
				require.NotContains(t, paths, "/api/admin.users.setAdmin")
				js := g.lastJSON(t)
				require.Equal(t, `"`+roleID+`"`, string(js["role_id"]))
			})
		}
	})
}
