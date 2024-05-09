package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	vault "github.com/hashicorp/vault/api"
)

type CfgManager interface {
	GetValue(ctx context.Context, key string) (string, error)
	GetValueOfDomainService(ctx context.Context, domain string, service string, key string) (string, error)
}

type ConfigManager struct {
	environment string
	domain      string
	service     string
	store       *vault.KVv2
}

func NewConfigManager() (CfgManager, error) {
	cfg := ConfigManager{
		environment: os.Getenv("ENVIRONMENT"),
		domain:      os.Getenv("DOMAIN"),
		service:     os.Getenv("SERVICE"),
	}
	config := &vault.Config{Address: os.Getenv("VAULT_ADDR")}

	err := config.ConfigureTLS(&vault.TLSConfig{Insecure: true})

	if err != nil {
		return nil, err
	}

	client, err := vault.NewClient(config)
	if err != nil {
		log.Fatalf("unable to initialize Vault client: %v", err)
	}

	// Authenticate
	// WARNING: This quickstart uses the root token for our Vault dev server.
	// Don't do this in production!
	client.SetToken(os.Getenv("VAULT_TOKEN"))

	store := client.KVv2("secrets")
	cfg.store = store
	return cfg, nil
}

func (cfg ConfigManager) GetValue(ctx context.Context, key string) (string, error) {
	return cfg.GetValueOfDomainService(ctx, cfg.domain, cfg.service, key)
}

func (cfg ConfigManager) GetValueOfDomainService(ctx context.Context, domain string, service string, key string) (string, error) {
	secret, err := cfg.store.Get(ctx, fmt.Sprintf("%s/%s/%s/%s", cfg.environment, domain, service, key))
	if err != nil {
		return "", err
	}
	if v, ok := secret.Data["value"]; ok {
		return v.(string), nil
	}
	return "", errors.New("secret_key_not_found")
}
