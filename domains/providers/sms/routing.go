package sms

import (
	"encoding/json"
	"fmt"

	"github.com/qeetgroup/qeet-notify/domains/routing"
)

// BuildProviders instantiates SMS providers from decrypted routing records,
// preserving priority order.
func BuildProviders(records []routing.Record) ([]Provider, error) {
	providers := make([]Provider, 0, len(records))
	for _, r := range records {
		p, err := buildProvider(r)
		if err != nil {
			return nil, fmt.Errorf("sms routing: %s: %w", r.ProviderName, err)
		}
		providers = append(providers, p)
	}
	return providers, nil
}

type msg91Config struct {
	APIKey string `json:"api_key"`
}

type twoFactorConfig struct {
	APIKey string `json:"api_key"`
}

func buildProvider(r routing.Record) (Provider, error) {
	switch r.ProviderName {
	case "msg91":
		var cfg msg91Config
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			return nil, fmt.Errorf("decode msg91 config: %w", err)
		}
		return NewMSG91(cfg.APIKey), nil
	case "2factor":
		var cfg twoFactorConfig
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			return nil, fmt.Errorf("decode 2factor config: %w", err)
		}
		return NewTwoFactor(cfg.APIKey), nil
	default:
		return nil, fmt.Errorf("unknown sms provider: %q", r.ProviderName)
	}
}
