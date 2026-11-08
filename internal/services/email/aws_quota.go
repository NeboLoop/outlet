package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
)

// SESQuota holds AWS SES sending quota information
type SESQuota struct {
	Max24HourSend   float64 `json:"max_24_hour_send"`   // Max emails in 24hr period
	MaxSendRate     float64 `json:"max_send_rate"`      // Max emails per second
	SentLast24Hours float64 `json:"sent_last_24_hours"` // Emails sent in last 24hrs
	RemainingQuota  float64 `json:"remaining_quota"`    // Remaining quota for today
}

// GetSESQuota queries AWS SES for the current sending quota
// If accessKey and secretKey are empty, uses default AWS credential chain
func GetSESQuota(ctx context.Context, region, accessKey, secretKey string) (*SESQuota, error) {
	if region == "" {
		region = "us-east-1" // Default region
	}

	var cfg aws.Config
	var err error

	if accessKey != "" && secretKey != "" {
		// Use provided credentials
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
	} else {
		// Use default credential chain (env vars, IAM role, etc.)
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SES client
	client := ses.NewFromConfig(cfg)

	// Get send quota
	result, err := client.GetSendQuota(ctx, &ses.GetSendQuotaInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get SES quota: %w", err)
	}

	quota := &SESQuota{
		Max24HourSend:   result.Max24HourSend,
		MaxSendRate:     result.MaxSendRate,
		SentLast24Hours: result.SentLast24Hours,
		RemainingQuota:  result.Max24HourSend - result.SentLast24Hours,
	}

	return quota, nil
}

// DetectAndSaveQuota detects AWS SES quota and returns config suitable for saving
func DetectAndSaveQuota(ctx context.Context, region, accessKey, secretKey string) (*OrgEmailConfig, error) {
	quota, err := GetSESQuota(ctx, region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	config := &OrgEmailConfig{
		SESRateLimit:  quota.MaxSendRate,
		SESRateBurst:  int(quota.MaxSendRate * 2), // Allow 2 seconds worth as burst
		SESDailyQuota: int64(quota.Max24HourSend),
	}

	return config, nil
}
