package enterprise

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/session"
	"github.com/conductorone/baton-sdk/pkg/types/sessions"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/slack-go/slack"
)

const (
	teamIDKey         = "team_id"
	userIDKey         = "user_id"
	scimPatchOpSchema = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	PageSizeDefault   = 100

	// Slack API error string constants.
	// SlackErrNoSuchSubteam is returned by the Slack API for user groups. We determined
	// empirically that retrying on this error resolves the issue.
	SlackErrNoSuchSubteam = "no_such_subteam"

	ScimVersionV2 = "v2"
	ScimVersionV1 = "v1"
)

var workspaceNameNamespace = sessions.WithPrefix("workspace_name")

type Client struct {
	baseScimUrl  *url.URL
	baseUrl      *url.URL
	token        string
	enterpriseID string
	botToken     string
	scimVersion  string
	wrapper      *uhttp.BaseHttpClient
}

func NewClient(
	httpClient *http.Client,
	token string,
	botToken string,
	enterpriseID string,
	govEnv bool,
	baseURLOverride string,
) (*Client, error) {
	finalBaseUrl := baseUrl
	finalBaseScimUrl := baseScimUrl
	finalScimVersion := ScimVersionV2
	// Custom base URL takes precedence over gov environment
	if baseURLOverride != "" {
		finalBaseUrl = baseURLOverride
		finalBaseScimUrl = baseURLOverride
	} else if govEnv {
		finalBaseUrl = baseGovUrl
		finalBaseScimUrl = baseGovScimUrl
		finalScimVersion = ScimVersionV1
	}

	baseUrl0, err := url.Parse(finalBaseUrl)
	if err != nil {
		return nil, err
	}

	baseScimUrl0, err := url.Parse(finalBaseScimUrl)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseUrl:      baseUrl0,
		baseScimUrl:  baseScimUrl0,
		token:        token,
		enterpriseID: enterpriseID,
		botToken:     botToken,
		scimVersion:  finalScimVersion,
		wrapper:      uhttp.NewBaseHttpClient(httpClient),
	}, nil
}

func (c *Client) SetWorkspaceNames(ctx context.Context, ss sessions.SessionStore, workspaces []slack.Team) error {
	workspaceMap := make(map[string]string)
	for _, workspace := range workspaces {
		workspaceMap[workspace.ID] = workspace.Name
	}
	return session.SetManyJSON(ctx, ss, workspaceMap, workspaceNameNamespace)
}

// GetWorkspaceNames retrieves workspace names for the given IDs from the session store.
func (c *Client) GetWorkspaceNames(ctx context.Context, ss sessions.SessionStore, workspaceIDs []string) (map[string]string, []string, error) {
	validIDs := make([]string, 0, len(workspaceIDs))
	for _, id := range workspaceIDs {
		if id != "" {
			validIDs = append(validIDs, id)
		}
	}

	if len(validIDs) == 0 {
		return make(map[string]string), []string{}, nil
	}

	found, err := session.GetManyJSON[string](ctx, ss, validIDs, workspaceNameNamespace)
	if err != nil {
		return nil, nil, err
	}

	missing := make([]string, 0)
	for _, id := range validIDs {
		if _, exists := found[id]; !exists {
			missing = append(missing, id)
		}
	}

	return found, missing, nil
}

// GetUserInfo returns the user info for the given user ID.
func (c *Client) GetUserInfo(
	ctx context.Context,
	userID string,
) (
	*User,
	*v2.RateLimitDescription,
	error,
) {
	var response struct {
		BaseResponse
		User *User `json:"user"`
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathGetUserInfo,
		&response,
		map[string]interface{}{"user": userID},
		true,
	)
	if err != nil {
		return nil, ratelimitData, err
	}

	return response.User, ratelimitData, nil
}

// GetUserGroupMembers returns the members of the given user group from a given team.
func (c *Client) GetUserGroupMembers(
	ctx context.Context,
	userGroupID string,
	teamID string,
) (
	[]string,
	*v2.RateLimitDescription,
	error,
) {
	var response struct {
		BaseResponse
		Users []string `json:"users"`
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathGetUserGroupMembers,
		&response,
		map[string]interface{}{
			teamIDKey:   teamID,
			"usergroup": userGroupID,
		},
		true,
	)
	if err != nil {
		return nil, ratelimitData, err
	}

	return response.Users, ratelimitData, nil
}

// UpdateUserGroupMembers replaces the entire list of users in a user group.
// https://docs.slack.dev/reference/methods/usergroups.users.update
func (c *Client) UpdateUserGroupMembers(
	ctx context.Context,
	userGroupID string,
	teamID string,
	users []string,
) (
	*v2.RateLimitDescription,
	error,
) {
	var response BaseResponse

	ratelimitData, err := c.post(
		ctx,
		UrlPathUpdateUserGroupMembers,
		&response,
		map[string]interface{}{
			"usergroup": userGroupID,
			"team_id":   teamID,
			"users":     strings.Join(users, ","),
		},
		false,
	)
	if err != nil {
		return ratelimitData, err
	}

	return ratelimitData, nil
}

// GetUsersAdmin returns all users in Enterprise grid.
func (c *Client) GetUsersAdmin(
	ctx context.Context,
	cursor string,
) (
	[]UserAdmin,
	string,
	*v2.RateLimitDescription,
	error,
) {
	values := map[string]interface{}{}

	// We need to check if cursor is empty because API throws error if empty string is passed.
	if cursor != "" {
		values["cursor"] = cursor
	}

	var response struct {
		BaseResponse
		Users []UserAdmin `json:"users"`
		Pagination
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathGetUsersAdmin,
		&response,
		values,
		false,
	)
	if err != nil {
		return nil, "", ratelimitData, err
	}

	nextToken := response.ResponseMetadata.NextCursor
	return response.Users, nextToken, ratelimitData, nil
}

// GetUsers returns the users of the given team.
func (c *Client) GetUsers(
	ctx context.Context,
	teamID string,
	cursor string,
) (
	[]User,
	string,
	*v2.RateLimitDescription,
	error,
) {
	values := map[string]interface{}{teamIDKey: teamID}

	// need to check if cursor is empty because API throws error if empty string is passed
	if cursor != "" {
		values["cursor"] = cursor
	}

	var response struct {
		BaseResponse
		Users []User `json:"members"`
		Pagination
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathGetUsers,
		&response,
		values,
		true,
	)
	if err != nil {
		return nil, "", ratelimitData, err
	}

	return response.Users,
		response.ResponseMetadata.NextCursor,
		ratelimitData,
		nil
}

// GetTeams returns the teams of the given enterprise.
func (c *Client) GetTeams(
	ctx context.Context,
	cursor string,
) (
	[]slack.Team,
	string,
	*v2.RateLimitDescription,
	error,
) {
	values := map[string]interface{}{}

	if cursor != "" {
		values["cursor"] = cursor
	}

	var response struct {
		BaseResponse
		Teams []slack.Team `json:"teams"`
		Pagination
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathGetTeams,
		&response,
		values,
		false,
	)
	if err != nil {
		return nil, "", ratelimitData, err
	}

	return response.Teams,
		response.ResponseMetadata.NextCursor,
		ratelimitData,
		nil
}

// GetRoleAssignments returns the role assignments for the given role ID.
func (c *Client) GetRoleAssignments(
	ctx context.Context,
	roleID string,
	cursor string,
) (
	[]RoleAssignment,
	string,
	*v2.RateLimitDescription,
	error,
) {
	values := map[string]interface{}{}

	if roleID != "" {
		values["role_ids"] = roleID
	}

	if cursor != "" {
		values["cursor"] = cursor
	}

	var response struct {
		BaseResponse
		RoleAssignments []RoleAssignment `json:"role_assignments"`
		Pagination
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathGetRoleAssignments,
		&response,
		values,
		false,
	)
	if err != nil {
		return nil, "", ratelimitData, err
	}

	return response.RoleAssignments,
		response.ResponseMetadata.NextCursor,
		ratelimitData,
		nil
}

// GetUserGroups returns the user groups for the given team.
func (c *Client) GetUserGroups(
	ctx context.Context,
	teamID string,
) (
	[]slack.UserGroup,
	*v2.RateLimitDescription,
	error,
) {
	var response struct {
		BaseResponse
		UserGroups []slack.UserGroup `json:"usergroups"`
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathGetUserGroups,
		&response,
		map[string]interface{}{teamIDKey: teamID},
		true,
	)
	if err != nil {
		return nil, ratelimitData, err
	}

	return response.UserGroups, ratelimitData, nil
}

// GetAuthTeamsList returns the list of teams for which the app is authed.
func (c *Client) GetAuthTeamsList(
	ctx context.Context,
	cursor string,
) (
	[]slack.Team,
	string,
	*v2.RateLimitDescription,
	error,
) {
	values := map[string]interface{}{}

	if cursor != "" {
		values["cursor"] = cursor
	}

	var response struct {
		BaseResponse
		Teams []slack.Team `json:"teams"`
		Pagination
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathAuthTeamsList,
		&response,
		values,
		false,
	)
	if err != nil {
		return nil, "", ratelimitData, err
	}

	return response.Teams,
		response.ResponseMetadata.NextCursor,
		ratelimitData,
		nil
}

// SetWorkspaceRole sets the role for the given user in the given team.
func (c *Client) SetWorkspaceRole(
	ctx context.Context,
	teamID string,
	userID string,
	roleID string,
) (
	*v2.RateLimitDescription,
	error,
) {
	actionUrl, err := getWorkspaceUrlPathByRole(roleID)
	if err != nil {
		return nil, err
	}

	var response BaseResponse

	ratelimitData, err := c.post(
		ctx,
		actionUrl,
		&response,
		map[string]interface{}{
			teamIDKey: teamID,
			userIDKey: userID,
		},
		false,
	)
	if err != nil {
		return ratelimitData, err
	}
	return ratelimitData, nil
}

// ListIDPGroups returns all IDP groups from the SCIM API.
func (c *Client) ListIDPGroups(
	ctx context.Context,
	startIndex int,
	count int,
) (
	*SCIMResponse[GroupResource],
	*v2.RateLimitDescription,
	error,
) {
	var response SCIMResponse[GroupResource]
	urlPathIDPGroups := fmt.Sprintf(UrlPathIDPGroups, c.scimVersion)
	ratelimitData, err := c.doRequest(
		ctx,
		http.MethodGet,
		c.getUrl(urlPathIDPGroups, map[string]interface{}{
			"startIndex": startIndex,
			"count":      count,
		}, true),
		&response,
		WithBearerToken(c.token),
	)
	if err != nil {
		return nil, ratelimitData, fmt.Errorf("error fetching IDP groups: %w", err)
	}

	return &response, ratelimitData, nil
}

// GetIDPGroup returns a single IDP group from the SCIM API.
func (c *Client) GetIDPGroup(
	ctx context.Context,
	groupID string,
) (
	*GroupResource,
	*v2.RateLimitDescription,
	error,
) {
	var response GroupResource
	ratelimitData, err := c.doRequest(
		ctx,
		http.MethodGet,
		c.getUrl(fmt.Sprintf(UrlPathIDPGroup, c.scimVersion, groupID), nil, true),
		&response,
		WithBearerToken(c.token),
	)
	if err != nil {
		return nil, ratelimitData, fmt.Errorf("error fetching IDP group: %w", err)
	}

	return &response, ratelimitData, nil
}

// ListIDPUsers returns all IDP users from the SCIM API.
func (c *Client) ListIDPUsers(
	ctx context.Context,
	startIndex int,
	count int,
) (
	*SCIMResponse[UserResource],
	*v2.RateLimitDescription,
	error,
) {
	var response SCIMResponse[UserResource]
	urlPathIDPUsers := fmt.Sprintf(UrlPathIDPUsers, c.scimVersion)
	ratelimitData, err := c.doRequest(
		ctx,
		http.MethodGet,
		c.getUrl(urlPathIDPUsers, map[string]interface{}{
			"startIndex": startIndex,
			"count":      count,
		}, true),
		&response,
		WithBearerToken(c.token),
	)
	if err != nil {
		return nil, ratelimitData, fmt.Errorf("error fetching IDP users: %w", err)
	}

	return &response, ratelimitData, nil
}

// AddUserToGroup patches a group by adding a user to it.
func (c *Client) AddUserToGroup(
	ctx context.Context,
	groupID string,
	user string,
) (
	*v2.RateLimitDescription,
	error,
) {
	requestBody := PatchOp{
		Schemas: []string{scimPatchOpSchema},
		Operations: []ScimOperate{
			{
				Op:   "add",
				Path: "members",
				Value: []UserID{
					{Value: user},
				},
			},
		},
	}

	ratelimitData, err := c.patchGroup(ctx, groupID, requestBody)
	if err != nil {
		return ratelimitData, fmt.Errorf("error adding user to IDP group: %w", err)
	}

	return ratelimitData, nil
}

// RemoveUserFromGroup patches a group by removing a user from it.
func (c *Client) RemoveUserFromGroup(
	ctx context.Context,
	groupID string,
	user string,
) (
	bool,
	*v2.RateLimitDescription,
	error,
) {
	// First, we need to fetch group to get existing members.
	group, ratelimitData, err := c.GetIDPGroup(ctx, groupID)
	if err != nil {
		return false, ratelimitData, fmt.Errorf("error fetching IDP group: %w", err)
	}

	found := false
	var result []UserID
	for _, member := range group.Members {
		if member.Value == user {
			found = true
		} else {
			result = append(result, UserID{Value: member.Value})
		}
	}

	// If we don't find the user, we can short-circuit here.
	if !found {
		return false, ratelimitData, nil
	}

	requestBody := PatchOp{
		Schemas: []string{scimPatchOpSchema},
		Operations: []ScimOperate{
			{
				Op:    "replace",
				Path:  "members",
				Value: result,
			},
		},
	}

	ratelimitData, err = c.patchGroup(ctx, groupID, requestBody)
	if err != nil {
		return false, ratelimitData, fmt.Errorf("error removing user from IDP group: %w", err)
	}

	return true, ratelimitData, nil
}

func (c *Client) patchGroup(
	ctx context.Context,
	groupID string,
	requestBody PatchOp,
) (
	*v2.RateLimitDescription,
	error,
) {
	var response *GroupResource
	ratelimitData, err := c.doRequest(
		ctx,
		http.MethodPatch,
		c.getUrl(fmt.Sprintf(UrlPathIDPGroup, c.scimVersion, groupID), nil, true),
		&response,
		WithBearerToken(c.token),
		uhttp.WithJSONBody(requestBody),
	)
	if err != nil {
		return ratelimitData, fmt.Errorf("error patching IDP group: %w", err)
	}

	return ratelimitData, nil
}

func (o *Client) AddUser(ctx context.Context, teamID, userID string) (*v2.RateLimitDescription, error) {
	var response BaseResponse
	ratelimitData, err := o.post(
		ctx,
		UrlPathUserAdd,
		&response,
		map[string]interface{}{
			teamIDKey: teamID,
			userIDKey: userID,
		},
		false,
	)

	if err != nil {
		return ratelimitData, err
	}

	return ratelimitData, nil
}

func (o *Client) RemoveUser(ctx context.Context, teamID, userID string) (*v2.RateLimitDescription, error) {
	var response BaseResponse
	ratelimitData, err := o.post(
		ctx,
		UrlPathUserRemove,
		&response,
		map[string]interface{}{
			teamIDKey: teamID,
			userIDKey: userID,
		},
		false,
	)

	if err != nil {
		return ratelimitData, err
	}

	return ratelimitData, nil
}

type InviteUserParams struct {
	TeamID     string
	ChannelIDs string
	Email      string
}

func (o *Client) InviteUserToWorkspace(ctx context.Context, p *InviteUserParams) (*v2.RateLimitDescription, error) {
	var response BaseResponse
	ratelimitData, err := o.post(
		ctx,
		UrlPathUserInvite,
		&response,
		map[string]interface{}{
			teamIDKey:     p.TeamID,
			"channel_ids": p.ChannelIDs,
			"email":       p.Email,
		},
		false,
	)
	if err != nil {
		return ratelimitData, err
	}
	return ratelimitData, nil
}

// DisableUser deactivates a user via SCIM API using DELETE.
// https://docs.slack.dev/reference/scim-api/
func (c *Client) DisableUser(
	ctx context.Context,
	userID string,
) (
	*v2.RateLimitDescription,
	error,
) {
	var emptyResponse interface{}
	ratelimitData, err := c.doRequest(
		ctx,
		http.MethodDelete,
		c.getUrl(fmt.Sprintf(UrlPathIDPUser, c.scimVersion, userID), nil, true),
		&emptyResponse,
		WithBearerToken(c.token),
	)
	if err != nil {
		return ratelimitData, fmt.Errorf("error disabling user: %w", err)
	}

	return ratelimitData, nil
}

// EnableUser activates a user via SCIM API by setting active to true.
func (c *Client) EnableUser(
	ctx context.Context,
	userID string,
) (
	*v2.RateLimitDescription,
	error,
) {
	requestBody := map[string]any{
		"schemas": []string{scimPatchOpSchema},
		"Operations": []map[string]any{
			{
				"path":  "active",
				"op":    "replace",
				"value": true,
			},
		},
	}

	var baseCheck struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}
	ratelimitData, err := c.doRequest(
		ctx,
		http.MethodPatch,
		c.getUrl(fmt.Sprintf(UrlPathIDPUser, c.scimVersion, userID), nil, true),
		&baseCheck,
		WithBearerToken(c.token),
		uhttp.WithJSONBody(requestBody),
	)
	if err != nil {
		return ratelimitData, fmt.Errorf("error enabling user: %w", err)
	}

	return ratelimitData, nil
}

func (c *Client) AssignEnterpriseRole(
	ctx context.Context,
	roleID string,
	userID string,
	teamID string,
) (
	*v2.RateLimitDescription,
	error,
) {
	if c.enterpriseID == "" {
		return nil, fmt.Errorf("enterprise ID is required for role assignment")
	}

	var response struct {
		BaseResponse
		RejectedUsers    []string `json:"rejected_users"`
		RejectedEntities []string `json:"rejected_entities"`
	}

	entityIDs := []string{teamID}
	params := map[string]interface{}{
		"role_id":    roleID,
		"user_ids":   []string{userID},
		"entity_ids": entityIDs,
	}

	ratelimitData, err := c.postJSON(
		ctx,
		UrlPathAssignEnterpriseRole,
		&response,
		params,
		false,
	)

	if err != nil {
		if len(response.RejectedUsers) > 0 || len(response.RejectedEntities) > 0 {
			return ratelimitData, fmt.Errorf("%w - rejected_users: %v, rejected_entities: %v", err, response.RejectedUsers, response.RejectedEntities)
		}
		return ratelimitData, err
	}
	return ratelimitData, nil
}

func (c *Client) UnassignEnterpriseRole(
	ctx context.Context,
	roleID string,
	userID string,
	teamID string,
) (
	*v2.RateLimitDescription,
	error,
) {
	if c.enterpriseID == "" {
		return nil, fmt.Errorf("enterprise ID is required for role removal")
	}

	var response struct {
		BaseResponse
		RejectedUsers    []string `json:"rejected_users"`
		RejectedEntities []string `json:"rejected_entities"`
	}

	entityIDs := []string{teamID}
	params := map[string]interface{}{
		"role_id":    roleID,
		"user_ids":   []string{userID},
		"entity_ids": entityIDs,
	}

	ratelimitData, err := c.post(
		ctx,
		UrlPathUnassignEnterpriseRole,
		&response,
		params,
		false,
	)

	if err != nil {
		if len(response.RejectedUsers) > 0 || len(response.RejectedEntities) > 0 {
			return ratelimitData, fmt.Errorf("%w - rejected_users: %v, rejected_entities: %v", err, response.RejectedUsers, response.RejectedEntities)
		}
		return ratelimitData, err
	}
	return ratelimitData, nil
}
