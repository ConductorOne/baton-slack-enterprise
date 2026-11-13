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
	// cannedUserInfo overrides the default /api/users.info response when set.
	cannedUserInfo []byte
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
		cannedErr := g.cannedError
		cannedUser := g.cannedUserInfo
		g.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/users.info" {
			if cannedUser != nil {
				_, _ = w.Write(cannedUser)
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"user":{"profile":{"team":"T123"}}}`))
			return
		}
		if cannedErr != nil {
			_, _ = w.Write(cannedErr)
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

// T3a/T3b: org-role revokes are per-arm (judge thread 4407347809). Revoking
// organization_owner demotes via admin.users.setAdmin with the org ID — never
// setRegular, which would strip BOTH org roles. Revoking organization_admin
// from a non-owner downgrades via admin.users.setRegular with the org ID.
// Neither path touches admin.roles.removeAssignments (AC 3, 6).
func TestEnterpriseRoleRevoke_OrgRolesDowngradeToRegular(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		roleID        string
		wantPath      string
		wantNoPath    string
		wantUsersInfo bool
	}{
		{OrganizationOwnerID, "/api/admin.users.setAdmin", "/api/admin.users.setRegular", true},
		{OrganizationAdminID, "/api/admin.users.setRegular", "/api/admin.users.setAdmin", true},
	} {
		t.Run(tc.roleID, func(t *testing.T) {
			g := newGuardServer(t)
			if tc.roleID == OrganizationOwnerID {
				// owner-revoke arm requires a real owner to reach setAdmin.
				g.mu.Lock()
				g.cannedUserInfo = []byte(`{"ok":true,"user":{"id":"U1","profile":{"team":"T123"},"enterprise_user":{"is_owner":true}}}`)
				g.mu.Unlock()
			}
			b := newGuardBuilder(t, g, guardTestOrgID)

			_, err := b.Revoke(ctx, guardRevokeInput(tc.roleID))
			require.NoError(t, err)

			paths := g.paths(t)
			require.Contains(t, paths, tc.wantPath)
			require.NotContains(t, paths, tc.wantNoPath)
			require.NotContains(t, paths, "/api/admin.roles.removeAssignments")
			if tc.wantUsersInfo {
				require.Contains(t, paths, "/api/users.info")
			} else {
				require.NotContains(t, paths, "/api/users.info")
			}
			form := g.lastForm(t)
			require.Equal(t, guardTestOrgID, form.Get("team_id"))
			require.Equal(t, "U1", form.Get("user_id"))
		})
	}
}

// T3c: revoking organization_admin from a user who is STILL an owner is
// refused fail-closed (FailedPrecondition, baton prefix) with ZERO downgrade
// calls — setRegular would strip the owner's ownership too (judge thread
// 4407347809).
func TestEnterpriseRoleRevoke_AdminOnStillOwnerRefused(t *testing.T) {
	ctx := context.Background()
	g := newGuardServer(t)
	g.mu.Lock()
	g.cannedUserInfo = []byte(`{"ok":true,"user":{"id":"U1","profile":{"team":"T123"},"enterprise_user":{"is_owner":true,"is_admin":true}}}`)
	g.mu.Unlock()
	b := newGuardBuilder(t, g, guardTestOrgID)

	ann, err := b.Revoke(ctx, guardRevokeInput(OrganizationAdminID))
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Contains(t, err.Error(), "baton-slack-enterprise:")
	require.Contains(t, err.Error(), "organization_owner")
	require.False(t, ann.Contains(&v2.GrantAlreadyRevoked{}))

	paths := g.paths(t)
	require.Contains(t, paths, "/api/users.info")
	require.NotContains(t, paths, "/api/admin.users.setRegular")
	require.NotContains(t, paths, "/api/admin.users.setAdmin")
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

// T7: idempotent revokes, per-arm (judge thread 4407347809). Owner-revoke arm
// (setAdmin): already_admin means the user is not an owner, so the owner role
// is already gone → GrantAlreadyRevoked. Admin-revoke arm (setRegular):
// already_regular yields GrantAlreadyRevoked; invalid_role_for_user is a
// REFUSAL and fails closed (deep-review F1, see TestEnterpriseRoleRevoke_AdminOnInvalidRoleFailClosed).
func TestEnterpriseRoleRevoke_IdempotentAlreadyRevoked(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		roleID   string
		slackErr string
	}{
		{OrganizationOwnerID, "already_admin"},
		{OrganizationAdminID, "already_regular"},
	} {
		t.Run(tc.roleID+"/"+tc.slackErr, func(t *testing.T) {
			g := newGuardServer(t)
			if tc.roleID == OrganizationOwnerID {
				// the owner-revoke arm pre-checks ownership before setAdmin.
				g.mu.Lock()
				g.cannedUserInfo = []byte(`{"ok":true,"user":{"id":"U1","profile":{"team":"T123"},"enterprise_user":{"is_owner":true}}}`)
				g.mu.Unlock()
			}
			g.mu.Lock()
			g.cannedError = []byte(`{"ok":false,"error":"` + tc.slackErr + `"}`)
			g.mu.Unlock()
			b := newGuardBuilder(t, g, guardTestOrgID)

			ann, err := b.Revoke(ctx, guardRevokeInput(tc.roleID))
			require.NoError(t, err)
			require.True(t, ann.Contains(&v2.GrantAlreadyRevoked{}))
		})
	}
}

// T7c (deep-review F1): invalid_role_for_user on admin.users.setRegular is a
// REFUSAL of the demotion ("the user's current role does not allow this
// change"), NOT an already-in-state signal — the admin-revoke arm must fail
// closed instead of reporting GrantAlreadyRevoked while the role persists.
func TestEnterpriseRoleRevoke_AdminOnInvalidRoleFailClosed(t *testing.T) {
	ctx := context.Background()
	g := newGuardServer(t)
	g.mu.Lock()
	g.cannedError = []byte(`{"ok":false,"error":"invalid_role_for_user"}`)
	g.mu.Unlock()
	b := newGuardBuilder(t, g, guardTestOrgID)

	ann, err := b.Revoke(ctx, guardRevokeInput(OrganizationAdminID))
	require.Error(t, err)
	require.False(t, ann.Contains(&v2.GrantAlreadyRevoked{}))
}

// T9 (deep-review F2): granting an org role to a current OWNER is realistic
// already-in-state (the read path emits BOTH org grants for every owner), so
// both grant org arms pre-check ownership via users.info and return
// GrantAlreadyExists with ZERO write calls — setAdmin on an owner would
// demote them, and setOwner on an owner relies on undocumented no-op
// behavior.
func TestEnterpriseRoleGrant_OwnerGrantOnOwnerAlreadyExists(t *testing.T) {
	ctx := context.Background()
	for _, roleID := range []string{OrganizationAdminID, OrganizationOwnerID} {
		t.Run(roleID, func(t *testing.T) {
			g := newGuardServer(t)
			g.mu.Lock()
			g.cannedUserInfo = []byte(`{"ok":true,"user":{"id":"U1","profile":{"team":"T123"},"enterprise_user":{"is_owner":true}}}`)
			g.mu.Unlock()
			b := newGuardBuilder(t, g, guardTestOrgID)

			principal, ent := guardGrantInput(roleID)
			ann, err := b.Grant(ctx, principal, ent)
			require.NoError(t, err)
			require.True(t, ann.Contains(&v2.GrantAlreadyExists{}))

			paths := g.paths(t)
			require.Contains(t, paths, "/api/users.info")
			require.NotContains(t, paths, "/api/admin.users.setAdmin")
			require.NotContains(t, paths, "/api/admin.users.setOwner")
		})
	}
}

// T7b: revoking organization_owner from a user who is NOT an owner (e.g.
// demoted outside C1) returns GrantAlreadyRevoked with ZERO write calls —
// setAdmin would promote them to org admin (judge thread 4407362910).
func TestEnterpriseRoleRevoke_OwnerOnNonOwnerAlreadyRevoked(t *testing.T) {
	ctx := context.Background()
	g := newGuardServer(t)
	// default canned /api/users.info has no enterprise_user → not an owner.
	b := newGuardBuilder(t, g, guardTestOrgID)

	ann, err := b.Revoke(ctx, guardRevokeInput(OrganizationOwnerID))
	require.NoError(t, err)
	require.True(t, ann.Contains(&v2.GrantAlreadyRevoked{}))

	paths := g.paths(t)
	require.Contains(t, paths, "/api/users.info")
	require.NotContains(t, paths, "/api/admin.users.setAdmin")
	require.NotContains(t, paths, "/api/admin.users.setRegular")
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
