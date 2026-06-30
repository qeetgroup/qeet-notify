package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	// Server
	HTTPPort string `envconfig:"HTTP_PORT" default:"8080"`
	Env      string `envconfig:"ENV" default:"development"`

	// Database
	DatabaseURL    string `envconfig:"DATABASE_URL" required:"true"`
	MigrationsDir  string `envconfig:"MIGRATIONS_DIR" default:"migrations"`

	// NATS
	NATSURL string `envconfig:"NATS_URL" default:"nats://localhost:4222"`

	// Redis
	RedisURL string `envconfig:"REDIS_URL" default:"redis://localhost:6379"`

	// S3 / Object Store
	S3Bucket   string `envconfig:"S3_BUCKET" default:""`
	S3Endpoint string `envconfig:"S3_ENDPOINT" default:""`     // MinIO in dev
	S3Region   string `envconfig:"S3_REGION" default:"ap-south-1"`

	// Auth
	QeetIDIssuer  string `envconfig:"QEET_ID_ISSUER" default:"https://api.id.qeet.in"`
	EncryptionKey string `envconfig:"ENCRYPTION_KEY" default:"dev-key-change-in-prod"` // pgp_sym_encrypt key

	// Email providers
	AWSSESRegion    string `envconfig:"AWS_SES_REGION" default:"ap-south-1"`
	AWSSESAccessKey string `envconfig:"AWS_SES_ACCESS_KEY_ID" default:""`
	AWSSESSecretKey string `envconfig:"AWS_SES_SECRET_ACCESS_KEY" default:""`
	ResendAPIKey    string `envconfig:"RESEND_API_KEY" default:""`

	// SMS providers
	MSG91APIKey   string `envconfig:"MSG91_API_KEY" default:""`
	TwoFactorKey  string `envconfig:"TWOFACTOR_API_KEY" default:""`

	// WhatsApp
	MetaWAToken       string `envconfig:"META_WHATSAPP_TOKEN" default:""`
	MetaWAPhoneID     string `envconfig:"META_WHATSAPP_PHONE_ID" default:""`
	MetaWAVerifyToken string `envconfig:"META_WHATSAPP_VERIFY_TOKEN" default:""`

	// Observability
	QeetLogsOTLPEndpoint string `envconfig:"QEET_LOGS_OTLP_ENDPOINT" default:""`
}

func Load() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}
	return &c, nil
}
