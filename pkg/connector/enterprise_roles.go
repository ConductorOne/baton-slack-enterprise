package connector

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resources "github.com/conductorone/baton-sdk/pkg/types/resource"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/conductorone/baton-slack-enterprise/pkg"
	enterprise "github.com/conductorone/baton-slack-enterprise/pkg/connector/client"
	"google.golang.org/grpc/codes"
)

const (
	// Enterprise grid system roles.
	AnalyticsAdmin         = "Rl0L"
	AuditLogsAdmin         = "Rl0C"
	ChannelAdmin           = "Rl01"
	ChannelManager         = "Rl0A"
	ConversationAdmin      = "Rl05"
	DLPAdmin               = "Rl09"
	ExportsAdmin           = "Rl0F"
	IntegrationsManager    = "Rl0D"
	MessageActivityManager = "Rl04"
	RoleAdmin              = "Rl02"
	SalesAdmin             = "Rl0G"
	SalesUser              = "Rl0H"
	SecurityAdmin          = "Rl0J"
	SlackPlatformDeveloper = "Rl0B"
	UserAdmin              = "Rl03"
	WorkflowAdmin          = "Rl0K"

	// Enterprise grid organization roles.
	OrganizationPrimaryOwnerID = "organization_primary_owner"
	OrganizationOwnerID        = "organization_owner"
	OrganizationAdminID        = "organization_admin"
)

var systemRoles = map[string]string{
	AnalyticsAdmin:         "Analytics Admin",
	AuditLogsAdmin:         "Audit Logs Admin",
	ChannelAdmin:           "Channel Admin",
	ChannelManager:         "Channel Manager",
	ConversationAdmin:      "Conversation Admin",
	DLPAdmin:               "DLP Admin",
	ExportsAdmin:           "Exports Admin",
	IntegrationsManager:    "Integrations Manager",
	MessageActivityManager: "Message Activity Manager",
	RoleAdmin:              "Role Admin",
	SalesAdmin:             "Sales Admin",
	SalesUser:              "Sales User",
	SecurityAdmin:          "Security Admin",
	SlackPlatformDeveloper: "Slack Platform Developer",
	UserAdmin:              "User Admin",
	WorkflowAdmin:          "Workflow Admin",
}

var organizationRoles = map[string]string{
	OrganizationPrimaryOwnerID: "Organization primary owner",
	OrganizationOwnerID:        "Organization owner",
	OrganizationAdminID:        "Organization admin",
}

type enterpriseRoleType struct {
	resourceType     *v2.ResourceType
	enterpriseClient *enterprise.Client
	enterpriseID     string
}

func (o *enterpriseRoleType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func enterpriseRoleBuilder(enterpriseID string, enterpriseClient *enterprise.Client) *enterpriseRoleType {
	return &enterpriseRoleType{
		resourceType:     resourceTypeEnterpriseRole,
		enterpriseClient: enterpriseClient,
		enterpriseID:     enterpriseID,
	}
}

func enterpriseRoleResource(
	_ context.Context,
	roleID string,
	_ *v2.ResourceId,
) (*v2.Resource, error) {
	var roleName string
	systemRoleName, ok := systemRoles[roleID]
	if !ok {
		orgRoleName, ok := organizationRoles[roleID]
		if !ok {
			return nil, fmt.Errorf("invalid system or organization roleID: %s", roleID)
		} else {
			roleName = orgRoleName
		}
	} else {
		roleName = systemRoleName
	}

	return resources.NewRoleResource(
		roleName,
		resourceTypeEnterpriseRole,
		roleID,
		nil,
	)
}

func (o *enterpriseRoleType) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	attrs resources.SyncOpAttrs,
) (
	[]*v2.Resource,
	*resources.SyncOpResults,
	error,
) {
	var ret []*v2.Resource

	bag, err := pkg.ParseRolesPageToken(attrs.PageToken.Token)
	if err != nil {
		return nil, nil, err
	}

	// We only want to do this once.
	if bag.Cursor == "" {
		for orgRoleID := range organizationRoles {
			r, err := enterpriseRoleResource(ctx, orgRoleID, parentResourceID)
			if err != nil {
				return nil, nil, err
			}

			ret = append(ret, r)
		}
	}

	outputAnnotations := annotations.New()
	roleAssignments, nextPage, ratelimitData, err := o.enterpriseClient.GetRoleAssignments(ctx, "", bag.Cursor)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resources.SyncOpResults{Annotations: outputAnnotations}, err
	}

	bag.Cursor = nextPage

	for _, roleAssignment := range roleAssignments {
		if _, ok := bag.FoundMap[roleAssignment.RoleID]; ok {
			continue
		}

		if _, ok := systemRoles[roleAssignment.RoleID]; !ok {
			continue
		}

		r, err := enterpriseRoleResource(ctx, roleAssignment.RoleID, parentResourceID)
		if err != nil {
			return nil, nil, err
		}

		ret = append(ret, r)

		bag.FoundMap[roleAssignment.RoleID] = true
	}

	nextPageToken, err := bag.Marshal()
	if err != nil {
		return nil, nil, err
	}

	return ret, &resources.SyncOpResults{NextPageToken: nextPageToken, Annotations: outputAnnotations}, nil
}

func (o *enterpriseRoleType) Entitlements(
	_ context.Context,
	resource *v2.Resource,
	_ resources.SyncOpAttrs,
) (
	[]*v2.Entitlement,
	*resources.SyncOpResults,
	error,
) {
	return []*v2.Entitlement{
			entitlement.NewAssignmentEntitlement(
				resource,
				RoleAssignmentEntitlement,
				entitlement.WithGrantableTo(resourceTypeUser),
				entitlement.WithDescription(
					fmt.Sprintf(
						"Has the %s role in the Slack enterprise",
						resource.DisplayName,
					),
				),
				entitlement.WithDisplayName(
					fmt.Sprintf(
						"%s Enterprise Role",
						resource.DisplayName,
					),
				),
			),
		},
		&resources.SyncOpResults{},
		nil
}

func (o *enterpriseRoleType) Grants(
	ctx context.Context,
	resource *v2.Resource,
	attrs resources.SyncOpAttrs,
) (
	[]*v2.Grant,
	*resources.SyncOpResults,
	error,
) {
	var rv []*v2.Grant

	bag, err := pkg.ParsePageToken(attrs.PageToken.Token, &v2.ResourceId{ResourceType: resourceTypeEnterpriseRole.Id})
	if err != nil {
		return nil, nil, err
	}

	// If current role is one of organization roles, don't return any grants
	// since we grant those on the user itself.
	if _, ok := organizationRoles[resource.Id.Resource]; ok {
		return nil, &resources.SyncOpResults{}, nil
	}

	outputAnnotations := annotations.New()
	roleAssignments, nextPage, ratelimitData, err := o.enterpriseClient.GetRoleAssignments(
		ctx,
		resource.Id.Resource,
		bag.PageToken(),
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resources.SyncOpResults{Annotations: outputAnnotations}, err
	}

	pageToken, err := bag.NextToken(nextPage)
	if err != nil {
		return nil, nil, err
	}

	for _, assignment := range roleAssignments {
		userID, err := resources.NewResourceID(resourceTypeUser, assignment.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create resourceID for user: %w", err)
		}

		rv = append(rv, grant.NewGrant(resource, RoleAssignmentEntitlement, userID))
	}

	return rv, &resources.SyncOpResults{NextPageToken: pageToken, Annotations: outputAnnotations}, nil
}

// orgRoleToWorkspaceRole maps an organization role ID to the workspace role
// word accepted by admin.users.setAdmin/setOwner. Note that "" (RegularRoleID)
// resolves to admin.users.setRegular via getWorkspaceUrlPathByRole; revoke
// callers pass RegularRoleID explicitly to downgrade either org role.
func orgRoleToWorkspaceRole(roleID string) string {
	switch roleID {
	case OrganizationAdminID:
		return AdminRoleID
	case OrganizationOwnerID:
		return OwnerRoleID
	default:
		return RegularRoleID
	}
}

// getTeamIDForUser retrieves the team ID for a user by calling the Slack API directly.
func (o *enterpriseRoleType) getTeamIDForUser(ctx context.Context, userID string) (string, *v2.RateLimitDescription, error) {
	user, ratelimitData, err := o.enterpriseClient.GetUserInfo(ctx, userID)
	if err != nil {
		return "", ratelimitData, fmt.Errorf("failed to get user info for user %s: %w", userID, err)
	}

	if user.Profile.Team != "" {
		return user.Profile.Team, ratelimitData, nil
	}

	return "", ratelimitData, fmt.Errorf("user %s has no team", userID)
}

func (o *enterpriseRoleType) Grant(
	ctx context.Context,
	principal *v2.Resource,
	entitlement *v2.Entitlement,
) (
	annotations.Annotations,
	error,
) {
	roleID := entitlement.Resource.Id.Resource
	userID := principal.Id.Resource

	outputAnnotations := annotations.New()

	if _, ok := organizationRoles[roleID]; ok {
		// Organization-level roles are granted via the org-level admin API
		// (admin.users.setAdmin/setOwner), not via admin.roles.addAssignments.
		// Guard order is locked: the empty-enterpriseID guard precedes the
		// primary-owner refusal, and both precede any HTTP call.
		if o.enterpriseID == "" {
			return outputAnnotations, uhttp.WrapErrors(
				codes.InvalidArgument,
				"baton-slack-enterprise: enterprise ID is required to grant or revoke organization-level roles",
				errors.New("connector was constructed without an enterprise ID"),
			)
		}

		if roleID == OrganizationPrimaryOwnerID {
			return outputAnnotations, uhttp.WrapErrors(
				codes.PermissionDenied,
				"baton-slack-enterprise: organization_primary_owner cannot be granted or revoked via the Slack API",
				errors.New("slack disallows modifying the primary owner"),
			)
		}

		// Order matters (GO-P-062/GO-A-018): WHERE (org, not the user's home team), WHO, WHAT (the role WORD, not the enterprise role ID).
		orgID := o.enterpriseID                             // WHERE — the org, not the user's home team
		userToSet := principal.Id.Resource                  // WHO
		workspaceRoleWord := orgRoleToWorkspaceRole(roleID) // WHAT — "admin"/"owner" — never organization_admin
		ratelimitData, err := o.enterpriseClient.SetWorkspaceRole(ctx, orgID, userToSet, workspaceRoleWord)
		outputAnnotations.WithRateLimiting(ratelimitData)
		if err != nil {
			if roleID == OrganizationAdminID && strings.Contains(err.Error(), "already_admin") {
				outputAnnotations.Append(&v2.GrantAlreadyExists{})
				return outputAnnotations, nil
			}
			if roleID == OrganizationOwnerID && strings.Contains(err.Error(), "already_owner") {
				outputAnnotations.Append(&v2.GrantAlreadyExists{})
				return outputAnnotations, nil
			}
			return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to set enterprise role %s for user %s: %w", roleID, userID, err)
		}

		return outputAnnotations, nil
	}

	// Get team ID by calling the API directly
	teamID, ratelimitData, err := o.getTeamIDForUser(ctx, userID)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, err
	}

	ratelimitData, err = o.enterpriseClient.AssignEnterpriseRole(ctx, roleID, userID, teamID)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("failed to assign enterprise role %s to user %s in team %s: %w", roleID, userID, teamID, err)
	}

	return outputAnnotations, nil
}

func (o *enterpriseRoleType) Revoke(
	ctx context.Context,
	grant *v2.Grant,
) (
	annotations.Annotations,
	error,
) {
	roleID := grant.Entitlement.Resource.Id.Resource
	userID := grant.Principal.Id.Resource

	outputAnnotations := annotations.New()

	if _, ok := organizationRoles[roleID]; ok {
		// Organization-level roles are revoked via the org-level admin API.
		// Guard order is locked: the empty-enterpriseID guard precedes the
		// primary-owner refusal, and both precede any HTTP call.
		if o.enterpriseID == "" {
			return outputAnnotations, uhttp.WrapErrors(
				codes.InvalidArgument,
				"baton-slack-enterprise: enterprise ID is required to grant or revoke organization-level roles",
				errors.New("connector was constructed without an enterprise ID"),
			)
		}

		if roleID == OrganizationPrimaryOwnerID {
			return outputAnnotations, uhttp.WrapErrors(
				codes.PermissionDenied,
				"baton-slack-enterprise: organization_primary_owner cannot be granted or revoked via the Slack API",
				errors.New("slack disallows modifying the primary owner"),
			)
		}

		// Order matters (GO-P-062/GO-A-018): WHERE (org, not the user's home team), WHO, WHAT (the role WORD, not the enterprise role ID).
		//
		// Revoke semantics are per-arm so a revoke strips only the requested
		// org role: admin.users.setRegular clears BOTH org roles at once, so
		// revoking organization_owner from an owner demotes them to admin
		// (setAdmin) instead — revoking organization_admin from a user who is
		// still an owner is refused fail-closed (the owner role must be
		// revoked first) rather than quietly stripping ownership too.
		orgID := o.enterpriseID                  // WHERE — the org, not the user's home team
		userToSet := grant.Principal.Id.Resource // WHO
		if roleID == OrganizationOwnerID {
			// WHAT — demote owner → admin, revoking ONLY the owner role.
			// Check ownership first: a user who is not an owner has no owner
			// role to revoke, and calling setAdmin on them would PROMOTE them
			// to org admin (e.g. after an out-of-band demotion).
			user, ratelimitData, err := o.enterpriseClient.GetUserInfo(ctx, userID)
			outputAnnotations.WithRateLimiting(ratelimitData)
			if err != nil {
				return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to get user info for user %s before revoking enterprise role %s: %w", userID, roleID, err)
			}
			if user == nil || !user.Enterprise.IsOwner {
				outputAnnotations.Append(&v2.GrantAlreadyRevoked{})
				return outputAnnotations, nil
			}

			ratelimitData, err = o.enterpriseClient.SetWorkspaceRole(ctx, orgID, userToSet, AdminRoleID)
			outputAnnotations.WithRateLimiting(ratelimitData)
			if err != nil {
				// already_admin means the user is not an owner, so the owner
				// role is already gone.
				if strings.Contains(err.Error(), "already_admin") {
					outputAnnotations.Append(&v2.GrantAlreadyRevoked{})
					return outputAnnotations, nil
				}
				return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to revoke enterprise role %s from user %s: %w", roleID, userID, err)
			}

			return outputAnnotations, nil
		}

		// organization_admin revoke: refuse while the user still holds ownership.
		user, ratelimitData, err := o.enterpriseClient.GetUserInfo(ctx, userID)
		outputAnnotations.WithRateLimiting(ratelimitData)
		if err != nil {
			return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to get user info for user %s before revoking enterprise role %s: %w", userID, roleID, err)
		}
		if user != nil && user.Enterprise.IsOwner {
			return outputAnnotations, uhttp.WrapErrors(
				codes.FailedPrecondition,
				"baton-slack-enterprise: cannot revoke organization_admin from user who is still an organization owner; revoke organization_owner first",
				fmt.Errorf("user %s still holds the organization owner role", userID),
			)
		}

		// WHAT — "" downgrades the admin org role via admin.users.setRegular (getWorkspaceUrlPathByRole's "" arm); never the org roleID
		ratelimitData, err = o.enterpriseClient.SetWorkspaceRole(ctx, orgID, userToSet, RegularRoleID)
		outputAnnotations.WithRateLimiting(ratelimitData)
		if err != nil {
			if strings.Contains(err.Error(), "already_regular") || strings.Contains(err.Error(), "invalid_role_for_user") {
				outputAnnotations.Append(&v2.GrantAlreadyRevoked{})
				return outputAnnotations, nil
			}
			return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to revoke enterprise role %s from user %s: %w", roleID, userID, err)
		}

		return outputAnnotations, nil
	}

	// Get team ID by calling the API directly
	teamID, ratelimitData, err := o.getTeamIDForUser(ctx, userID)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, err
	}

	ratelimitData, err = o.enterpriseClient.UnassignEnterpriseRole(ctx, roleID, userID, teamID)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("failed to unassign enterprise role %s from user %s in team %s: %w", roleID, userID, teamID, err)
	}

	return outputAnnotations, nil
}
