package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

func capabilityPermissions(perms ...string) *v2.CapabilityPermissions {
	cp := &v2.CapabilityPermissions{}
	for _, p := range perms {
		cp.Permissions = append(cp.Permissions, &v2.CapabilityPermission{Permission: p})
	}
	return cp
}

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: annotations.New(
			&v2.SkipEntitlementsAndGrants{},
			capabilityPermissions(
				// Bot Token Scopes
				"users:read",
				"users:read.email",
				"users.profile:read",
				// User Token Scopes
				"admin",
				"admin.users:read",
				"admin.users:write",
			),
		),
	}
	resourceTypeWorkspace = &v2.ResourceType{
		Id:          "workspace",
		DisplayName: "Workspace",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
		Annotations: annotations.New(
			capabilityPermissions(
				// Bot Token Scopes
				"team:read",
				"users:read",
				"users:read.email",
				"channels:join",
				"channels:read",
				"groups:read",
				// User Token Scopes
				"admin",
				"admin.teams:read",
				"admin.users:read",
				"admin.users:write",
			),
		),
	}
	resourceTypeUserGroup = &v2.ResourceType{
		Id:          "userGroup",
		DisplayName: "User Group",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
		Annotations: annotations.New(
			capabilityPermissions(
				// Bot Token Scopes
				"usergroups:read",
				"usergroups:write",
				"users:read",
				// User Token Scopes
				"admin.usergroups:read",
				"admin.usergroups:write",
			),
		),
	}
	resourceTypeGroup = &v2.ResourceType{
		Id:          "group",
		DisplayName: "IDP Group",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
		Annotations: annotations.New(
			capabilityPermissions(
				// User Token Scopes (SCIM API)
				"admin",
				"admin.users:write",
			),
		),
	}
	resourceTypeWorkspaceRole = &v2.ResourceType{
		Id:          "workspaceRole",
		DisplayName: "Workspace Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
		Annotations: annotations.New(
			capabilityPermissions(
				// User Token Scopes
				"admin",
				"admin.users:write",
			),
		),
	}
	resourceTypeEnterpriseRole = &v2.ResourceType{
		Id:          "enterpriseRole",
		DisplayName: "Enterprise Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
		Annotations: annotations.New(
			capabilityPermissions(
				// Bot Token Scopes
				"users:read",
				// User Token Scopes
				"admin",
				"admin.roles:read",
				"admin.roles:write",
			),
		),
	}
)
