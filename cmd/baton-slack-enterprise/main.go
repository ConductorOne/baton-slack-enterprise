package main

import (
	"context"

	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
	cfg "github.com/conductorone/baton-slack-enterprise/pkg/config"
	"github.com/conductorone/baton-slack-enterprise/pkg/connector"
)

var (
	connectorName = "baton-slack-enterprise"
	version       = "dev"
)

func main() {
	ctx := context.Background()
	config.RunConnector(ctx, connectorName, version, cfg.Configuration, connector.New,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.Slack{}),
		connectorrunner.WithSessionStoreEnabled())
}
