package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// SESConfig holds AWS SES configuration
type SESConfig struct {
	Region      string
	AccessKey   string
	SecretKey   string
	FromAddress string
	FromName    string
	ReplyTo     string
}

// SendEmailViaSES sends an email using the AWS SES API directly
// This is the preferred method when AWS credentials are configured
func SendEmailViaSES(ctx context.Context, sesConfig *SESConfig, to, subject, htmlBody string) error {
	client, err := newSESClient(ctx, sesConfig)
	if err != nil {
		return err
	}

	// Build the from address (a non-ASCII display name is RFC 2047 encoded)
	from := formatFrom(sesConfig.FromName, sesConfig.FromAddress)

	// Build reply-to addresses
	var replyToAddresses []string
	if sesConfig.ReplyTo != "" {
		replyToAddresses = []string{sesConfig.ReplyTo}
	}

	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(htmlBody),
				},
			},
			Subject: &types.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source:           aws.String(from),
		ReplyToAddresses: replyToAddresses,
	}

	_, err = client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("SES SendEmail failed: %w", err)
	}

	return nil
}

// newSESClient builds an SES client from the config, using static credentials
// when provided and otherwise the default AWS credential chain.
func newSESClient(ctx context.Context, sesConfig *SESConfig) (*ses.Client, error) {
	if sesConfig.Region == "" {
		sesConfig.Region = "us-east-1"
	}
	var cfg aws.Config
	var err error
	if sesConfig.AccessKey != "" && sesConfig.SecretKey != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(sesConfig.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				sesConfig.AccessKey, sesConfig.SecretKey, "",
			)),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(sesConfig.Region))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return ses.NewFromConfig(cfg), nil
}

// SendRawEmailViaSES sends a pre-built raw MIME message (used for attachments
// via multipart/mixed) through SES SendRawEmail.
func SendRawEmailViaSES(ctx context.Context, sesConfig *SESConfig, raw []byte) error {
	client, err := newSESClient(ctx, sesConfig)
	if err != nil {
		return err
	}
	input := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{Data: raw},
	}
	if sesConfig.FromAddress != "" {
		input.Source = aws.String(sesConfig.FromAddress)
	}
	if _, err := client.SendRawEmail(ctx, input); err != nil {
		return fmt.Errorf("SES SendRawEmail failed: %w", err)
	}
	return nil
}
