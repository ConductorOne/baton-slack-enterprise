package main

import (
	"github.com/conductorone/baton-sdk/pkg/config"
	cfg "github.com/conductorone/baton-slack-enterprise/pkg/config"
)

func main() {
	config.Generate("slack-enterprise", cfg.Configuration)
}
