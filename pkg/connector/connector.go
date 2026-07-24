package connector

import (
	"context"
	"fmt"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	cfg "github.com/conductorone/baton-slack-enterprise/pkg/config"
	enterprise "github.com/conductorone/baton-slack-enterprise/pkg/connector/client"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type Slack struct {
	client              *slack.Client
	apiKey              string
	enterpriseClient    *enterprise.Client
	enterpriseID        string
	govEnv              bool
	syncWorkspaceRoles  bool
	syncEnterpriseRoles bool
}

const govSlackApiUrl = "https://api.slack-gov.com/api/"

// Metadata returns metadata about the connector.
func (c *Slack) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Slack",
		Description: "Connector syncing users, workspaces, user groups and workspace roles from Slack to Baton.",
		AccountCreationSchema: &v2.ConnectorAccountCreationSchema{
			FieldMap: map[string]*v2.ConnectorAccountCreationSchema_Field{
				"channel_ids": {
					DisplayName: "ChannelIDs",
					Required:    true,
					Description: "Channel IDs the user will be invited to",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "ChannelIDs",
					Order:       1,
				},
				"email": {
					DisplayName: "Email",
					Required:    true,
					Description: "This email will be used as the login for the user.",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "Email",
					Order:       2,
				},
				"team_id": {
					DisplayName: "WorkspaceID",
					Required:    true,
					Description: "The workspaceID the user will be invited to",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "TeamID",
					Order:       3,
				},
			},
		},
	}, nil
}

// Validate hits the Slack API to validate that the authenticated user has needed permissions.
func (s *Slack) Validate(ctx context.Context) (annotations.Annotations, error) {
	res, err := s.client.AuthTestContext(ctx)
	if err != nil {
		return nil, enterprise.WrapError(err, "authenticating", nil)
	}

	user, err := s.client.GetUserInfoContext(ctx, res.UserID)
	if err != nil {
		return nil, enterprise.WrapError(err, "retrieving authenticated user", nil)
	}

	isValidUser := user.IsAdmin || user.IsOwner || user.IsPrimaryOwner || user.IsBot
	if !isValidUser {
		return nil, uhttp.WrapErrors(
			codes.PermissionDenied,
			"authenticated user is not an admin, owner, primary owner or a bot",
			fmt.Errorf("user lacks required permissions"),
		)
	}
	return nil, nil
}

type slackLogger struct {
	ZapLog *zap.Logger
}

// Output Needed to prevent slack client from logging in its own format.
func (s *slackLogger) Output(callDepth int, msg string) error {
	s.ZapLog.Info(msg, zap.Int("callDepth", callDepth))
	return nil
}

func NewSlack(ctx context.Context, apiKey, enterpriseKey string, govEnv bool, baseURL string) (*Slack, error) {
	l := ctxzap.Extract(ctx)
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, l))
	if err != nil {
		return nil, err
	}

	logger := &slackLogger{ZapLog: l}
	opts := []slack.Option{
		slack.OptionDebug(true),
		slack.OptionHTTPClient(httpClient),
		slack.OptionLog(logger),
	}
	// Custom base URL takes precedence over gov environment
	if baseURL != "" {
		opts = append(opts, slack.OptionAPIURL(baseURL))
	} else if govEnv {
		opts = append(opts, slack.OptionAPIURL(govSlackApiUrl))
	}
	client := slack.New(apiKey, opts...)

	res, err := client.AuthTestContext(ctx)
	if err != nil {
		return nil, enterprise.WrapError(err, "authenticating during initialization", nil)
	}

	if res.EnterpriseID == "" {
		return nil, uhttp.WrapErrors(
			codes.InvalidArgument,
			"baton-slack-enterprise requires a Slack Enterprise Grid account",
			fmt.Errorf("not an enterprise account"),
		)
	}

	if enterpriseKey == "" {
		return nil, uhttp.WrapErrors(
			codes.InvalidArgument,
			"enterprise token is required for baton-slack-enterprise",
			fmt.Errorf("missing enterprise token"),
		)
	}

	enterpriseId := res.EnterpriseID
	enterpriseClient, err := enterprise.NewClient(
		httpClient,
		enterpriseKey,
		apiKey,
		res.EnterpriseID,
		govEnv,
		baseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create enterprise client. Error: %w", err)
	}
	return &Slack{
		client:           client,
		apiKey:           apiKey,
		enterpriseClient: enterpriseClient,
		enterpriseID:     enterpriseId,
		govEnv:           govEnv,
	}, nil
}

func New(ctx context.Context, config *cfg.SlackEnterprise, opts *cli.ConnectorOpts) (connectorbuilder.ConnectorBuilderV2, []connectorbuilder.Opt, error) {
	cb, err := NewSlack(
		ctx,
		config.Token,
		config.EnterpriseToken,
		config.GovEnv,
		config.BaseUrl,
	)
	if err != nil {
		return nil, nil, err
	}

	// The workspace resource's Grants() also emits workspaceRole and enterpriseRole
	// grants as an optimization, since the underlying Slack API response for
	// workspace members already contains role info. Only emit those cross-type
	// grants if the caller is actually syncing those resource types.
	cb.syncWorkspaceRoles = true
	cb.syncEnterpriseRoles = true
	if opts != nil {
		cb.syncWorkspaceRoles = opts.WillSyncResourceType(resourceTypeWorkspaceRole.Id)
		cb.syncEnterpriseRoles = opts.WillSyncResourceType(resourceTypeEnterpriseRole.Id)
	}

	builderOpts := []connectorbuilder.Opt{}
	return cb, builderOpts, nil
}

func (s *Slack) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncerV2 {
	return []connectorbuilder.ResourceSyncerV2{
		userBuilder(s.client, s.enterpriseID, s.enterpriseClient),
		workspaceBuilder(s.client, s.enterpriseID, s.enterpriseClient, s.syncWorkspaceRoles, s.syncEnterpriseRoles),
		userGroupBuilder(s.enterpriseID, s.enterpriseClient),
		workspaceRoleBuilder(s.client, s.enterpriseID, s.enterpriseClient),
		enterpriseRoleBuilder(s.enterpriseID, s.enterpriseClient),
		groupBuilder(s.enterpriseClient, s.enterpriseID, s.govEnv),
	}
}
