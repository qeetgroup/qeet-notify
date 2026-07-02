package whatsapp

import (
	"encoding/json"
	"fmt"

	"github.com/qeetgroup/qeet-notify/domains/routing"
)

// BuildProviders instantiates WhatsApp providers from decrypted routing records,
// preserving priority order.
func BuildProviders(records []routing.Record) ([]Provider, error) {
	providers := make([]Provider, 0, len(records))
	for _, r := range records {
		p, err := buildProvider(r)
		if err != nil {
			return nil, fmt.Errorf("whatsapp routing: %s: %w", r.ProviderName, err)
		}
		providers = append(providers, p)
	}
	return providers, nil
}

type metaConfig struct {
	Token   string `json:"token"`
	PhoneID string `json:"phone_id"`
}

func buildProvider(r routing.Record) (Provider, error) {
	switch r.ProviderName {
	case "meta":
		var cfg metaConfig
		if err := json.Unmarshal(r.Config, &cfg); err != nil {
			return nil, fmt.Errorf("decode meta config: %w", err)
		}
		return NewMeta(cfg.Token, cfg.PhoneID), nil
	default:
		return nil, fmt.Errorf("unknown whatsapp provider: %q", r.ProviderName)
	}
}
