package connector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	resources "github.com/conductorone/baton-sdk/pkg/types/resource"
	enterprise "github.com/conductorone/baton-slack-enterprise/pkg/connector/client"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
)

// One workspace admin who is also an enterprise admin. Grants therefore emits
// two workspace_role grants (admin + member role), one enterprise_role grant,
// and the workspace's own member grant.
const guardUsersResponse = `{
  "ok": true,
  "members": [
    {
      "id": "U1",
      "team_id": "test-team",
      "is_admin": true,
      "enterprise_user": {"id": "E1", "is_admin": true}
    }
  ]
}`

func newGuardWorkspaceBuilder(t *testing.T, skipWorkspaceRole, skipEnterpriseRole bool) *workspaceResourceType {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(guardUsersResponse))
	}))
	t.Cleanup(srv.Close)

	c, err := enterprise.NewClient(srv.Client(), "token", "bot-token", "test-enterprise", false, srv.URL)
	require.NoError(t, err)

	return workspaceBuilder(&slack.Client{}, "test-enterprise", c, skipWorkspaceRole, skipEnterpriseRole)
}

func TestWorkspaceGrants_CrossTypeGuards(t *testing.T) {
	res, err := resources.NewResource("acme", resourceTypeWorkspace, testTeamResID)
	require.NoError(t, err)

	for _, tc := range []struct {
		name               string
		skipWorkspaceRole  bool
		skipEnterpriseRole bool
		wantWorkspaceRole  int
		wantEnterpriseRole int
	}{
		{"both synced", false, false, 2, 1},
		{"workspace_role filtered out", true, false, 0, 1},
		{"enterprise_role filtered out", false, true, 2, 0},
		{"both filtered out", true, true, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newGuardWorkspaceBuilder(t, tc.skipWorkspaceRole, tc.skipEnterpriseRole)

			grants, _, err := b.Grants(context.Background(), res, resources.SyncOpAttrs{})
			require.NoError(t, err)

			counts := map[string]int{}
			for _, g := range grants {
				counts[g.GetEntitlement().GetResource().GetId().GetResourceType()]++
			}

			require.Equal(t, tc.wantWorkspaceRole, counts[resourceTypeWorkspaceRole.Id], "workspace_role grants")
			require.Equal(t, tc.wantEnterpriseRole, counts[resourceTypeEnterpriseRole.Id], "enterprise_role grants")
			// The workspace member grant is this type's own and is never gated.
			require.Equal(t, 1, counts[resourceTypeWorkspace.Id], "workspace member grants")
		})
	}
}

// The workspace type owns a member entitlement, so the resource-type skip
// annotations must not be applied to it even when both targets are filtered.
func TestWorkspaceResourceType_NoSkipAnnotations(t *testing.T) {
	rt := newGuardWorkspaceBuilder(t, true, true).ResourceType(context.Background())
	for _, a := range rt.GetAnnotations() {
		require.False(t, a.MessageIs(&v2.SkipEntitlements{}))
		require.False(t, a.MessageIs(&v2.SkipEntitlementsAndGrants{}))
	}
}
