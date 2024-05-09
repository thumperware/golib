package test

import (
	"context"
)

type MockConfigManager struct {
	keyVal map[string]string
}

func NewConfigManager() MockConfigManager {
	return MockConfigManager{
		keyVal: make(map[string]string),
	}
}

func (cfg *MockConfigManager) WithKeyValue(key string, value string) *MockConfigManager {
	cfg.keyVal[key] = value
	return cfg
}

func (cfg *MockConfigManager) WithDomainServiceKeyValue(domain string, service string, key string, value string) *MockConfigManager {
	cfg.keyVal[domain+"."+service+"."+key] = value
	return cfg
}

func (cfg MockConfigManager) GetValue(ctx context.Context, key string) (string, error) {
	return cfg.keyVal[key], nil
}

func (cfg MockConfigManager) GetValueOfDomainService(ctx context.Context, domain string, service string, key string) (string, error) {
	return cfg.keyVal[domain+"."+service+"."+key], nil
}
