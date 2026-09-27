package email

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/crypto"
)

// AWSCredentials returns the AWS credentials an organization sends with:
// its own when it has them, else the platform's (platform settings,
// decrypted with the credential key).
func AWSCredentials(ctx context.Context, store *db.Store, cryptoSvc *crypto.Service, orgID string) (region, accessKey, secretKey string, err error) {
	emailConfig, err := GetOrgEmailConfig(ctx, store, orgID)
	if err == nil && emailConfig.HasOwnAWSCredentials() {
		return emailConfig.AWSRegion, emailConfig.AWSAccessKey, emailConfig.AWSSecretKey, nil
	}

	// Fall back to platform credentials
	awsSettings, err := store.GetPlatformSettingsByCategory(ctx, "aws")
	if err != nil {
		return "", "", "", err
	}

	region = "us-east-1" // default

	for _, setting := range awsSettings {
		switch setting.Key {
		case "aws_access_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				if cryptoSvc != nil {
					decrypted, decErr := cryptoSvc.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						return "", "", "", decErr
					}
					accessKey = decrypted
				}
			}
		case "aws_secret_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				if cryptoSvc != nil {
					decrypted, decErr := cryptoSvc.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						return "", "", "", decErr
					}
					secretKey = decrypted
				}
			}
		case "aws_region":
			if setting.ValueText.Valid && setting.ValueText.String != "" {
				region = setting.ValueText.String
			}
		}
	}

	if accessKey == "" || secretKey == "" {
		return "", "", "", sql.ErrNoRows
	}

	return region, accessKey, secretKey, nil
}
