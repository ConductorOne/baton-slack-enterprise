package connector

import (
	"context"
	"fmt"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"

	enterprise "github.com/conductorone/baton-slack-enterprise/pkg/connector/client"
	"github.com/slack-go/slack"
)

type userGroupResourceType struct {
	resourceType     *v2.ResourceType
	enterpriseID     string
	enterpriseClient *enterprise.Client
}

func (o *userGroupResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func userGroupBuilder(
	enterpriseID string,
	enterpriseClient *enterprise.Client,
) *userGroupResourceType {
	return &userGroupResourceType{
		resourceType:     resourceTypeUserGroup,
		enterpriseID:     enterpriseID,
		enterpriseClient: enterpriseClient,
	}
}

// Create a new connector resource for a Slack user group.
func userGroupResource(
	ctx context.Context,
	userGroup slack.UserGroup,
	parentResourceID *v2.ResourceId,
) (*v2.Resource, error) {
	return resource.NewGroupResource(
		userGroup.Name,
		resourceTypeUserGroup,
		userGroup.ID,
		[]resource.GroupTraitOption{
			resource.WithGroupProfile(
				map[string]interface{}{
					"userGroup_id":     userGroup.ID,
					"userGroup_name":   userGroup.Name,
					"userGroup_handle": userGroup.Handle,
				},
			),
		},
		resource.WithParentResourceID(parentResourceID),
	)
}

func (o *userGroupResourceType) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	_ resource.SyncOpAttrs,
) (
	[]*v2.Resource,
	*resource.SyncOpResults,
	error,
) {
	if parentResourceID == nil {
		return nil, &resource.SyncOpResults{}, nil
	}

	var (
		userGroups    []slack.UserGroup
		ratelimitData *v2.RateLimitDescription
		err           error
	)
	outputAnnotations := annotations.New()
	// We use different method here because we need to pass a teamID, but it's
	// not supported by the slack-go library.
	userGroups, ratelimitData, err = o.enterpriseClient.GetUserGroups(ctx, parentResourceID.Resource)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: outputAnnotations}, err
	}

	output := make([]*v2.Resource, 0, len(userGroups))
	for _, ug := range userGroups {
		r, err := userGroupResource(ctx, ug, parentResourceID)
		if err != nil {
			return nil, nil, fmt.Errorf("creating user group resource: %w", err)
		}
		output = append(output, r)
	}
	return output, &resource.SyncOpResults{Annotations: outputAnnotations}, nil
}

func (o *userGroupResourceType) Entitlements(
	_ context.Context,
	res *v2.Resource,
	_ resource.SyncOpAttrs,
) (
	[]*v2.Entitlement,
	*resource.SyncOpResults,
	error,
) {
	return []*v2.Entitlement{
			entitlement.NewAssignmentEntitlement(
				res,
				memberEntitlement,
				entitlement.WithGrantableTo(resourceTypeUser),
				entitlement.WithDescription(
					fmt.Sprintf(
						"Member of %s User group",
						res.DisplayName,
					),
				),
				entitlement.WithDisplayName(
					fmt.Sprintf(
						"%s User group %s",
						res.DisplayName,
						memberEntitlement,
					),
				),
			),
		},
		&resource.SyncOpResults{},
		nil
}

func (o *userGroupResourceType) Grants(
	ctx context.Context,
	res *v2.Resource,
	_ resource.SyncOpAttrs,
) (
	[]*v2.Grant,
	*resource.SyncOpResults,
	error,
) {
	outputAnnotations := annotations.New()
	// TODO(marcos): This should use 2D pagination.
	groupMembers, ratelimitData, err := o.enterpriseClient.GetUserGroupMembers(
		ctx,
		res.Id.Resource,
		res.ParentResourceId.Resource,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resource.SyncOpResults{Annotations: outputAnnotations}, err
	}

	var rv []*v2.Grant
	for _, member := range groupMembers {
		userID, err := resource.NewResourceID(resourceTypeUser, member)
		if err != nil {
			return nil, &resource.SyncOpResults{Annotations: outputAnnotations}, fmt.Errorf("baton-slack-enterprise: failed to create resource ID for user group member %q: %w", member, err)
		}

		rv = append(rv, grant.NewGrant(res, memberEntitlement, userID))
	}

	return rv, &resource.SyncOpResults{Annotations: outputAnnotations}, nil
}

func (o *userGroupResourceType) Grant(
	ctx context.Context,
	principal *v2.Resource,
	ent *v2.Entitlement,
) (
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != resourceTypeUser.Id {
		logger.Warn(
			"baton-slack-enterprise: only users can be added to a user group",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-slack-enterprise: only users can be added to a user group")
	}

	userGroupID := ent.Resource.Id.Resource
	teamID := ent.Resource.ParentResourceId.Resource
	userID := principal.Id.Resource

	outputAnnotations := annotations.New()

	currentMembers, ratelimitData, err := o.enterpriseClient.GetUserGroupMembers(ctx, userGroupID, teamID)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to get user group members: %w", err)
	}

	for _, member := range currentMembers {
		if member == userID {
			outputAnnotations.Append(&v2.GrantAlreadyExists{})
			return outputAnnotations, nil
		}
	}

	currentMembers = append(currentMembers, userID)
	ratelimitData, err = o.enterpriseClient.UpdateUserGroupMembers(ctx, userGroupID, teamID, currentMembers)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to add user to user group: %w", err)
	}

	return outputAnnotations, nil
}

func (o *userGroupResourceType) Revoke(
	ctx context.Context,
	g *v2.Grant,
) (
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)

	principal := g.Principal
	ent := g.Entitlement

	if principal.Id.ResourceType != resourceTypeUser.Id {
		logger.Warn(
			"baton-slack-enterprise: only users can be removed from a user group",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-slack-enterprise: only users can be removed from a user group")
	}

	userGroupID := ent.Resource.Id.Resource
	teamID := ent.Resource.ParentResourceId.Resource
	userID := principal.Id.Resource

	outputAnnotations := annotations.New()

	currentMembers, ratelimitData, err := o.enterpriseClient.GetUserGroupMembers(ctx, userGroupID, teamID)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to get user group members: %w", err)
	}

	found := false
	newMembers := make([]string, 0, len(currentMembers))
	for _, member := range currentMembers {
		if member == userID {
			found = true
		} else {
			newMembers = append(newMembers, member)
		}
	}

	if !found {
		outputAnnotations.Append(&v2.GrantAlreadyRevoked{})
		return outputAnnotations, nil
	}

	ratelimitData, err = o.enterpriseClient.UpdateUserGroupMembers(ctx, userGroupID, teamID, newMembers)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return outputAnnotations, fmt.Errorf("baton-slack-enterprise: failed to remove user from user group: %w", err)
	}

	return outputAnnotations, nil
}
