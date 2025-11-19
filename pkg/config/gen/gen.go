package main

import (
	cfg "github.com/conductorone/baton-slack-enterprise/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("slack-enterprise", cfg.Config)
}
