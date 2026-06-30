package messaging

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
)

// DefaultMaxDeliver is the number of delivery attempts a channel job gets before
// it is dead-lettered. Channel consumers should set ConsumerConfig.MaxDeliver to
// the same value so the server stops redelivering once this many attempts fail.
const DefaultMaxDeliver = 5

// HandleFailure decides what to do with a message whose handler returned an
// error. Until it has been delivered maxDeliver times it is nak'd for another
// attempt; on the final attempt the original payload + error are published to
// the tenant's dead-letter subject and the message is terminated (no further
// redelivery, so a poison message can never block the queue forever).
func HandleFailure(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg, maxDeliver int, handlerErr error, log zerolog.Logger) {
	delivered := 1
	if md, err := msg.Metadata(); err == nil {
		delivered = int(md.NumDelivered)
	}
	if delivered < maxDeliver {
		msg.Nak() //nolint:errcheck
		return
	}

	errMsg := "unknown error"
	if handlerErr != nil {
		errMsg = handlerErr.Error()
	}
	entry, _ := json.Marshal(map[string]any{
		"subject":   msg.Subject(),
		"error":     errMsg,
		"delivered": delivered,
		"data":      string(msg.Data()),
	})
	if _, err := js.Publish(ctx, DLQSubject(TenantFromSubject(msg.Subject())), entry); err != nil {
		// Could not dead-letter — nak so the message is not silently lost.
		log.Error().Err(err).Str("subject", msg.Subject()).Msg("publish to DLQ failed")
		msg.Nak() //nolint:errcheck
		return
	}
	log.Warn().Str("subject", msg.Subject()).Int("delivered", delivered).Str("reason", errMsg).Msg("message dead-lettered")
	msg.Term() //nolint:errcheck
}
