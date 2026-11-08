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


// newSESClient builds an SES client from the config.
func newSESClient(ctx context.Context, sesConfig *SESConfig) (*ses.Client, error) {
	cfg, err := awsConfig(ctx, sesConfig)
	if err != nil {
		return nil, err
	}
	return ses.NewFromConfig(cfg), nil
}

// awsConfig is the AWS configuration for an SES account, using static
// credentials when provided and otherwise the default AWS credential chain.
func awsConfig(ctx context.Context, sesConfig *SESConfig) (aws.Config, error) {
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
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return cfg, nil
}

// SendRawEmailViaSES sends a complete RFC 5322 message through SES
// SendRawEmail. Destinations is set explicitly, so SES delivers to the one
// intended recipient and never to addresses parsed from the message headers.
func SendRawEmailViaSES(ctx context.Context, sesConfig *SESConfig, source, to string, raw []byte) error {
	client, err := newSESClient(ctx, sesConfig)
	if err != nil {
		return err
	}
	input := &ses.SendRawEmailInput{
		RawMessage:   &types.RawMessage{Data: raw},
		Source:       aws.String(source),
		Destinations: []string{to},
	}
	if _, err := client.SendRawEmail(ctx, input); err != nil {
		return fmt.Errorf("SES SendRawEmail failed: %w", err)
	}
	return nil
}
