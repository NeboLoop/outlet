package settings

import (
	"context"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPlatformSESQuotaLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPlatformSESQuotaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlatformSESQuotaLogic {
	return &GetPlatformSESQuotaLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPlatformSESQuotaLogic) GetPlatformSESQuota() (resp *types.SESQuotaResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Get AWS credentials from platform settings
	awsSettings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "aws")
	if err != nil {
		l.Error("failed to get AWS platform settings:", err)
		return nil, errorx.NewInternalError("failed to retrieve AWS settings")
	}

	var accessKey, secretKey, region string

	for _, setting := range awsSettings {
		switch setting.Key {
		case "aws_access_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				// Decrypt the access key
				if l.svcCtx.CryptoService != nil {
					decrypted, decErr := l.svcCtx.CryptoService.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						l.Error("failed to decrypt aws_access_key:", decErr)
						return nil, errorx.NewInternalError("failed to decrypt AWS credentials")
					}
					accessKey = decrypted
				}
			}
		case "aws_secret_key":
			if setting.ValueEncrypted.Valid && setting.ValueEncrypted.String != "" {
				// Decrypt the secret key
				if l.svcCtx.CryptoService != nil {
					decrypted, decErr := l.svcCtx.CryptoService.DecryptString([]byte(setting.ValueEncrypted.String))
					if decErr != nil {
						l.Error("failed to decrypt aws_secret_key:", decErr)
						return nil, errorx.NewInternalError("failed to decrypt AWS credentials")
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

	// If no credentials configured, return an error
	if accessKey == "" || secretKey == "" {
		return nil, errorx.NewBadRequestError("AWS credentials not configured. Please configure AWS SES in settings.")
	}

	// Default region if not set
	if region == "" {
		region = "us-east-1"
	}

	// Get the SES quota from AWS
	quota, err := email.GetSESQuota(l.ctx, region, accessKey, secretKey)
	if err != nil {
		l.Error("failed to get SES quota:", err)
		return nil, errorx.NewInternalError(fmt.Sprintf("failed to get SES quota: %v", err))
	}

	// Get timezone from platform settings (default to UTC)
	timezone := "UTC"
	tzSetting, err := l.svcCtx.DB.GetPlatformSetting(l.ctx, "timezone")
	if err == nil && tzSetting.ValueText.Valid && tzSetting.ValueText.String != "" {
		timezone = tzSetting.ValueText.String
	}

	return &types.SESQuotaResponse{
		Max24HourSend:   quota.Max24HourSend,
		MaxSendRate:     quota.MaxSendRate,
		SentLast24Hours: quota.SentLast24Hours,
		RemainingQuota:  quota.RemainingQuota,
		Region:          region,
		Timezone:        timezone,
	}, nil
}
