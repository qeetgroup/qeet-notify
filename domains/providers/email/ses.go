package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SESProvider struct {
	client *sesv2.Client
}

func NewSES(region, accessKey, secretKey string) (*SESProvider, error) {
	cfg, err := awscfg.LoadDefaultConfig(context.Background(),
		awscfg.WithRegion(region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("ses config: %w", err)
	}
	return &SESProvider{client: sesv2.NewFromConfig(cfg)}, nil
}

func (p *SESProvider) Name() string { return "ses" }

func (p *SESProvider) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	from := msg.From
	if msg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", msg.FromName, msg.From)
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination: &types.Destination{
			ToAddresses: []string{msg.To},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(msg.Subject)},
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(msg.HTMLBody)},
				},
			},
		},
	}
	if msg.TextBody != "" {
		input.Content.Simple.Body.Text = &types.Content{Data: aws.String(msg.TextBody)}
	}
	if msg.ReplyTo != "" {
		input.ReplyToAddresses = []string{msg.ReplyTo}
	}

	out, err := p.client.SendEmail(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("ses send: %w", err)
	}
	return &SendResult{ProviderMessageID: aws.ToString(out.MessageId)}, nil
}
