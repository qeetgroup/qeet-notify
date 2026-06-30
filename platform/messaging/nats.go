package messaging

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Client struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

func New(url string) (*Client, error) {
	nc, err := nats.Connect(url,
		nats.Name("qeet-notify-api"),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}
	return &Client{Conn: nc, JS: js}, nil
}

func (c *Client) Close() {
	c.Conn.Drain() //nolint:errcheck
}
