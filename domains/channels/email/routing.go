package email

import (
	"encoding/json"
	"fmt"

	"github.com/qeetgroup/qeet-notify/domains/routing"
)

// BuildProviders instantiates email providers from decrypted routing records,
// preserving priority order. Callers use routing.Load to fetch the records.
func BuildProviders(records []routing.Record) ([]Provider, error) {
	providers := make([]Provider, 0, len(records))
	for _, r := range records {
		p, err := buildProvider(r)
		if err != nil {
			return nil, fmt.Errorf("email routing: %s: %w", r.ProviderName, err)
		}
		providers = append(providers, p)
	}
	return providers, nil
}

type sesConfig struct {
	Region    string `json:"region"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type resendConfig struct {
	APIKey string `json:"api_key"`
}

func buildProvider(r routing.Record) (Provider, error) {
	switch r.ProviderName {
	case "ses":
		var cfg sesConfig
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			return nil, fmt.Errorf("decode ses config: %w", err)
		}
		return NewSES(cfg.Region, cfg.AccessKey, cfg.SecretKey)
	case "resend":
		var cfg resendConfig
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			return nil, fmt.Errorf("decode resend config: %w", err)
		}
		return NewResend(cfg.APIKey), nil
	default:
		return nil, fmt.Errorf("unknown email provider: %q", r.ProviderName)
	}
}
