package config_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thumperq/golib/config"
)

func TestNewConfigManager(t *testing.T) {
	//t.Skip("skipping test")
	ctx := context.Background()
	os.Setenv("VAULT_ADDR", "https://34.168.165.181:8200")
	os.Setenv("VAULT_TOKEN", "s.rCO7OV2nfAz10ocGATOJcDDn")
	os.Setenv("ENVIRONMENT", "dev")
	os.Setenv("DOMAIN", "wms")
	os.Setenv("SERVICE", "mailbox")
	cfg := config.NewConfigManager()
	value, err := cfg.GetValue(ctx, "DATABASE_URL")
	require.NoError(t, err)
	require.NotEmpty(t, value)
}
