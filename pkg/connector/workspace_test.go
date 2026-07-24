package connector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	resources "github.com/conductorone/baton-sdk/pkg/types/resource"
	enterprise "github.com/conductorone/baton-slack-enterprise/pkg/connector/client"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// Helper function to create a test builder with mocks.
func newTestWorkspaceBuilder() (*workspaceResourceType, *enterprise.MockSlackEnterpriseService) {
	return newTestWorkspaceBuilderWithSyncFlags(true, true)
}

// newTestWorkspaceBuilderWithSyncFlags is like newTestWorkspaceBuilder but lets
// callers control whether workspaceRole/enterpriseRole cross-type grants are
// enabled, to exercise the Grants() gating behavior.
func newTestWorkspaceBuilderWithSyncFlags(syncWorkspaceRoles, syncEnterpriseRoles bool) (*workspaceResourceType, *enterprise.MockSlackEnterpriseService) {
	mockEnterpriseClient := enterprise.Client{}
	mockSlackClient := slack.Client{}
	mockEnterpriseService := &enterprise.MockSlackEnterpriseService{}

	builder := workspaceBuilder(&mockSlackClient, "test-enterprise", &mockEnterpriseClient, syncWorkspaceRoles, syncEnterpriseRoles)

	// Replace the Enterprise service with our mock.
	builder.enterpriseService = mockEnterpriseService

	return builder, mockEnterpriseService
}

func TestWorkspaceGrantAndRevoke(t *testing.T) {
	ctx := context.Background()

	t.Run("Grant operation for workspace with valid principal and entitlement", func(t *testing.T) {
		workspaceBuilder, mockService := newTestWorkspaceBuilder()
		// Mock the add user to workspace call.
		mockService.AddUserFunc = func(ctx context.Context, teamID, userID string) (*v2.RateLimitDescription, error) {
			assert.Equal(t, testTeamResID, teamID)
			assert.Equal(t, testUserResID, userID)
			return nil, nil
		}

		// Create a grant request.
		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: resourceTypeUser.Id,
				Resource:     testUserResID,
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: testTeamResID,
				},
			},
		}

		// Execute Grant.
		_, err := workspaceBuilder.Grant(ctx, principal, entitlement)

		// Verify the result.
		require.NoError(t, err)
	})

	t.Run("Grant operation for workspace with invalid principal", func(t *testing.T) {
		workspaceBuilder, _ := newTestWorkspaceBuilder()

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: "invalid-type",
				Resource:     testUserResID,
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: testTeamResID,
				},
			},
		}

		// Execute Grant.
		_, err := workspaceBuilder.Grant(ctx, principal, entitlement)

		// Verify the error.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only users can be assigned to a workspace")
	})

	t.Run("Grant operation for workspace with already granted principal", func(t *testing.T) {
		workspaceBuilder, mockService := newTestWorkspaceBuilder()
		// Mock the add user to workspace call.
		mockService.AddUserFunc = func(ctx context.Context, teamID, userID string) (*v2.RateLimitDescription, error) {
			return nil, errors.New(codes.AlreadyExists.String())
		}

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: resourceTypeUser.Id,
				Resource:     testUserResID,
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: testTeamResID,
				},
			},
		}

		// Execute Grant.
		annotations, err := workspaceBuilder.Grant(ctx, principal, entitlement)

		// Verify the result.
		require.NoError(t, err)
		require.True(t, annotations.Contains(&v2.GrantAlreadyExists{}))
	})

	t.Run("Revoke operation for workspace with valid principal and entitlement", func(t *testing.T) {
		workspaceBuilder, mockService := newTestWorkspaceBuilder()
		// Mock the remove user from workspace call.
		mockService.RemoveUserFunc = func(ctx context.Context, teamID, userID string) (*v2.RateLimitDescription, error) {
			assert.Equal(t, testTeamResID, teamID)
			assert.Equal(t, testUserResID, userID)
			return nil, nil
		}

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: resourceTypeUser.Id,
				Resource:     testUserResID,
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: testTeamResID,
				},
			},
		}

		grant := &v2.Grant{
			Principal:   principal,
			Entitlement: entitlement,
		}

		// Execute Revoke.
		_, err := workspaceBuilder.Revoke(ctx, grant)

		// Verify the result.
		require.NoError(t, err)
	})

	t.Run("Revoke operation for workspace with invalid principal", func(t *testing.T) {
		workspaceBuilder, _ := newTestWorkspaceBuilder()

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: "invalid-type",
				Resource:     testUserResID,
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: testTeamResID,
				},
			},
		}

		grant := &v2.Grant{
			Principal:   principal,
			Entitlement: entitlement,
		}

		// Execute Revoke.
		_, err := workspaceBuilder.Revoke(ctx, grant)

		// Verify the error.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only users can be revoked from a workspace")
	})

	t.Run("Revoke operation for workspace with already revoked principal", func(t *testing.T) {
		workspaceBuilder, mockService := newTestWorkspaceBuilder()
		// Mock the remove user from workspace call.
		mockService.RemoveUserFunc = func(ctx context.Context, teamID, userID string) (*v2.RateLimitDescription, error) {
			return nil, errors.New(codes.NotFound.String())
		}

		principal := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: resourceTypeUser.Id,
				Resource:     testUserResID,
			},
		}

		entitlement := &v2.Entitlement{
			Resource: &v2.Resource{
				Id: &v2.ResourceId{
					Resource: testTeamResID,
				},
			},
		}

		grant := &v2.Grant{
			Principal:   principal,
			Entitlement: entitlement,
		}

		// Execute Revoke.
		annotations, err := workspaceBuilder.Revoke(ctx, grant)

		// Verify the result.
		require.NoError(t, err)
		require.True(t, annotations.Contains(&v2.GrantAlreadyRevoked{}))
	})
}

// TestWorkspaceGrantsCrossTypeFiltering verifies that workspaceResourceType.Grants
// gates its cross-type workspaceRole/enterpriseRole grant emission on the
// syncWorkspaceRoles/syncEnterpriseRoles flags, while always emitting the
// workspace's own member grant regardless of those flags.
func TestWorkspaceGrantsCrossTypeFiltering(t *testing.T) {
	ctx := context.Background()

	// A single synthetic user that trips exactly one workspace-role branch
	// (IsBot -> BotRoleID; the plain-member fallthrough requires !IsBot, so
	// it doesn't also fire) and one enterprise-role branch
	// (Enterprise.IsAdmin -> OrganizationAdminID) simultaneously, so a single
	// page exercises both cross-type emission paths with predictable counts.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			enterprise.BaseResponse
			enterprise.Pagination
			Members []enterprise.User `json:"members"`
		}{
			BaseResponse: enterprise.BaseResponse{Ok: true},
			Members: []enterprise.User{
				{
					ID:     "U1",
					TeamID: "T1",
					IsBot:  true,
					Enterprise: enterprise.EnterpriseUser{
						IsAdmin: true,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	newBuilderWithServer := func(syncWorkspaceRoles, syncEnterpriseRoles bool) *workspaceResourceType {
		client, err := enterprise.NewClient(http.DefaultClient, "test-token", "test-bot-token", "test-enterprise", false, server.URL)
		require.NoError(t, err)
		return workspaceBuilder(&slack.Client{}, "test-enterprise", client, syncWorkspaceRoles, syncEnterpriseRoles)
	}

	workspaceRes := &v2.Resource{
		Id: &v2.ResourceId{
			ResourceType: resourceTypeWorkspace.Id,
			Resource:     "T1",
		},
	}

	countGrantsByType := func(grants []*v2.Grant, resourceTypeID string) int {
		count := 0
		for _, g := range grants {
			if g.Entitlement.Resource.Id.ResourceType == resourceTypeID {
				count++
			}
		}
		return count
	}

	t.Run("emits cross-type grants when both flags enabled", func(t *testing.T) {
		builder := newBuilderWithServer(true, true)
		grants, _, err := builder.Grants(ctx, workspaceRes, resources.SyncOpAttrs{})
		require.NoError(t, err)

		assert.Equal(t, 1, countGrantsByType(grants, resourceTypeWorkspaceRole.Id))
		assert.Equal(t, 1, countGrantsByType(grants, resourceTypeEnterpriseRole.Id))
		// The workspace's own member grant must always be present.
		assert.Equal(t, 1, countGrantsByType(grants, resourceTypeWorkspace.Id))
	})

	t.Run("suppresses cross-type grants when both flags disabled", func(t *testing.T) {
		builder := newBuilderWithServer(false, false)
		grants, _, err := builder.Grants(ctx, workspaceRes, resources.SyncOpAttrs{})
		require.NoError(t, err)

		assert.Equal(t, 0, countGrantsByType(grants, resourceTypeWorkspaceRole.Id))
		assert.Equal(t, 0, countGrantsByType(grants, resourceTypeEnterpriseRole.Id))
		// The workspace's own member grant must still always be present.
		assert.Equal(t, 1, countGrantsByType(grants, resourceTypeWorkspace.Id))
	})

	t.Run("independently gates workspaceRole and enterpriseRole", func(t *testing.T) {
		builder := newBuilderWithServer(true, false)
		grants, _, err := builder.Grants(ctx, workspaceRes, resources.SyncOpAttrs{})
		require.NoError(t, err)

		assert.Equal(t, 1, countGrantsByType(grants, resourceTypeWorkspaceRole.Id))
		assert.Equal(t, 0, countGrantsByType(grants, resourceTypeEnterpriseRole.Id))

		builder2 := newBuilderWithServer(false, true)
		grants2, _, err := builder2.Grants(ctx, workspaceRes, resources.SyncOpAttrs{})
		require.NoError(t, err)

		assert.Equal(t, 0, countGrantsByType(grants2, resourceTypeWorkspaceRole.Id))
		assert.Equal(t, 1, countGrantsByType(grants2, resourceTypeEnterpriseRole.Id))
	})
}
