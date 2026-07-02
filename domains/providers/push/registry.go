package push

import (
	"fmt"

	"github.com/qeetgroup/qeet-notify/domains/routing"
)

// BuildProviders instantiates push providers from decrypted routing records.
// TODO: add fcm and apns cases once those sub-packages are implemented.
func BuildProviders(records []routing.Record) ([]Provider, error) {
	providers := make([]Provider, 0, len(records))
	for _, r := range records {
		return nil, fmt.Errorf("push routing: unknown provider %q (fcm/apns not yet implemented)", r.ProviderName)
	}
	return providers, nil
}
