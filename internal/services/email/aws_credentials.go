package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/crypto"
)

// AWSCredentials returns the AWS credentials a sending pathway uses: an ses
// pathway's own, the platform's for a platform pathway, and the platform's
// when pathwayID is empty (platform settings, decrypted with the credential
// key). A pathway that doesn't send through SES returns an error.
func AWSCredentials(ctx context.Context, store *db.Store, cryptoSvc *crypto.Service, orgID, pathwayID string) (region, accessKey, secretKey string, err error) {
	if pathwayID != "" {
		p, err := store.GetSendingPathway(ctx, db.GetSendingPathwayParams{ID: pathwayID, OrgID: orgID})
		if err != nil {
			return "", "", "", err
		}
		switch p.Provider {
		case ProviderSES:
			return pathwayAWSKeys(cryptoSvc, p)
		case ProviderPlatform:
		default:
			return "", "", "", fmt.Errorf("pathway %q sends through %s, not SES: %w", p.Name, p.Provider, sql.ErrNoRows)
		}
	}

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

// pathwayAWSKeys decrypts an ses pathway's own keys.
func pathwayAWSKeys(cryptoSvc *crypto.Service, p db.SendingPathway) (region, accessKey, secretKey string, err error) {
	if len(p.AwsAccessKeyEnc) == 0 || len(p.AwsSecretKeyEnc) == 0 {
		return "", "", "", fmt.Errorf("pathway %q has no AWS keys: %w", p.Name, sql.ErrNoRows)
	}
	if cryptoSvc == nil {
		return "", "", "", errors.New("ENCRYPTION_KEY is not set, so the pathway's AWS keys can't be read")
	}
	if accessKey, err = cryptoSvc.DecryptString(p.AwsAccessKeyEnc); err != nil {
		return "", "", "", err
	}
	if secretKey, err = cryptoSvc.DecryptString(p.AwsSecretKeyEnc); err != nil {
		return "", "", "", err
	}
	return firstNonEmpty(p.AwsRegion, "us-east-1"), accessKey, secretKey, nil
}
