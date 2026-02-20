package config

import (
	"testing"

	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/stretchr/testify/assert"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *SlackEnterprise
		wantErr bool
	}{
		{
			name: "valid config",
			config: &SlackEnterprise{ //nolint:gosec // Not real credentials, test values only
				Token:           "xoxb-test-token",
				EnterpriseToken: "xoxp-test-enterprise-token",
			},
			wantErr: false,
		},
		{
			name:    "invalid config - missing required fields",
			config:  &SlackEnterprise{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := field.Validate(Configuration, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
