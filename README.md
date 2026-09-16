`baton-slack-enterprise` is a connector for Slack built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the Slack API to sync data about workspaces, users, user groups, and channels.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Prerequisites

1. Create a Slack app. You can follow [this Slack quickstart guide](https://api.slack.com/authentication/basics).
2. Set needed Bot Token Scopes for the app: 
  - channels:join
  - channels:read
  - groups:read
  - team:read
  - usergroups:read
  - users.profile:read
  - users:read
  - users:read.email
3. Install the app to your workspace.
4. Use Bot User OAuth Token as token in `baton-slack-enterprise`.

For the enterprise grid plan the same rules apply for creating an app. There is 
a difference in setting scopes, for applications that will be installed on 
organization level, User Token Scopes should be set as well as bot scopes. User 
Token is used for Admin API needed to sync additional resources in the enterprise.
Additional scopes for User Token are:
  - admin
  - admin.roles:read
  - admin.teams:read
  - admin.usergroups:read
  - admin.users:read

For provisioning you will need this scope.
  - admin.users:write

5. **Permissions for User Activation/Deactivation Actions**: To use the enable_user and disable_user actions, you need:
  - An Enterprise Grid token with admin permissions in order to access the SCIM API
  - See [Slack SCIM API Permissions](https://docs.slack.dev/admins/scim-api/#permissions) for more details 

Other difference is in the way the application is installed, on enterprise grid 
app should be installed on the Organization level and on all the Workspaces from 
which you want to sync the resources. The installation has to be done by Admin 
or Owner of an Enterprise Grid organization. More info with an example is 
available in the [Slack API Docs](https://api.slack.com/methods/admin.teams.list#markdown).
To work with Enterprise Grid APIs use User OAuth Token passed as 
`--enterprise-token` along with the Bot User OAuth Token passed via `--token` flag.
To work with GovSlack instances use `--gov-env` flag along with the `--enterprise-token`.

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-slack-enterprise
baton-slack-enterprise
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_TOKEN=token public.ecr.aws/conductorone/baton-slack-enterprise:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-slack-enterprise/cmd/baton-slack-enterprise@main

BATON_TOKEN=token
baton resources
```

# Data Model

`baton-slack-enterprise` pulls down information about the following Slack resources:
- Workspaces
- Users
- User Groups
- Channels
- Workspace roles

Enterprise grid additional resources:
- Enterprise roles

# Contributing, Support, and Issues

We started Baton because we were tired of taking screenshots and manually 
building spreadsheets. We welcome contributions, and ideas, no matter how 
small&mdash;our goal is to make identity and permissions sprawl less painful for 
everyone. If you have questions, problems, or ideas: Please open a GitHub Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-slack-enterprise` Command Line Usage

```
baton-slack-enterprise

Usage:
  baton-slack-enterprise [flags]
  baton-slack-enterprise [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  config             Get the connector config schema
  help               Help about any command

Flags:
      --auth-method string                               ($BATON_AUTH_METHOD)
      --client-id string                                 The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string                             The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
      --enterprise-token string                          required: The Slack user oauth token used to connect to the Slack Enterprise Grid Admin API ($BATON_ENTERPRISE_TOKEN)
      --external-resource-c1z string                     The path to the c1z file to sync external baton resources with ($BATON_EXTERNAL_RESOURCE_C1Z)
      --external-resource-entitlement-id-filter string   The entitlement that external users, groups must have access to sync external baton resources ($BATON_EXTERNAL_RESOURCE_ENTITLEMENT_ID_FILTER)
  -f, --file string                                      The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
      --gov-env                                          Flag indicating to use Slack-Gov environment. ($BATON_GOV_ENV)
  -h, --help                                             help for baton-slack-enterprise
      --log-format string                                The output format for logs: json, console ($BATON_LOG_FORMAT) (default "console")
      --log-level string                                 The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
      --log-level-debug-expires-at string                The timestamp indicating when debug-level logging should expire ($BATON_LOG_LEVEL_DEBUG_EXPIRES_AT)
      --otel-collector-endpoint string                   The endpoint of the OpenTelemetry collector to send observability data to (used for both tracing and logging if specific endpoints are not provided) ($BATON_OTEL_COLLECTOR_ENDPOINT)
  -p, --provisioning                                     This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-entitlements-and-grants                     This must be set to skip syncing of entitlements and grants ($BATON_SKIP_ENTITLEMENTS_AND_GRANTS)
      --skip-full-sync                                   This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --sync-resource-types strings                      The resource type IDs to sync ($BATON_SYNC_RESOURCE_TYPES)
      --sync-resources strings                           The resource IDs to sync ($BATON_SYNC_RESOURCES)
      --ticketing                                        This must be set to enable ticketing support ($BATON_TICKETING)
      --token string                                     required: The Slack bot user oauth token used to connect to the Slack API ($BATON_TOKEN)
  -v, --version                                          version for baton-slack-enterprise

Use "baton-slack-enterprise [command] --help" for more information about a command.
```
