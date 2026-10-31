package emailconfig

import (
	"context"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOrgEmailConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateOrgEmailConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOrgEmailConfigLogic {
	return &UpdateOrgEmailConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOrgEmailConfigLogic) UpdateOrgEmailConfig(req *types.UpdateOrgEmailConfigRequest) (resp *types.OrgEmailConfigInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.OrgId, access.Admin); err != nil {
		return nil, err
	}

	// Get existing config first
	config, err := email.GetOrgEmailConfig(l.ctx, l.svcCtx.DB, req.OrgId)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing config: %w", err)
	}

	// Update only non-zero values from request
	if req.SESRateLimit > 0 {
		config.SESRateLimit = req.SESRateLimit
	}
	if req.SESRateBurst > 0 {
		config.SESRateBurst = req.SESRateBurst
	}
	if req.SESDailyQuota > 0 {
		config.SESDailyQuota = req.SESDailyQuota
	}
	if req.WorkerCount > 0 {
		config.WorkerCount = req.WorkerCount
	}
	if req.BatchSize > 0 {
		config.BatchSize = req.BatchSize
	}
	if req.AWSRegion != "" {
		config.AWSRegion = req.AWSRegion
	}
	if req.AWSAccessKey != "" {
		config.AWSAccessKey = req.AWSAccessKey
	}
	if req.AWSSecretKey != "" {
		config.AWSSecretKey = req.AWSSecretKey
	}
	if req.FromEmail != "" {
		config.FromEmail = req.FromEmail
	}
	if req.FromName != "" {
		config.FromName = req.FromName
	}
	if req.ReplyTo != "" {
		config.ReplyTo = req.ReplyTo
	}

	// Save updated config
	if err := email.SaveOrgEmailConfig(l.ctx, l.svcCtx.DB, req.OrgId, config); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &types.OrgEmailConfigInfo{
		SESRateLimit:  config.SESRateLimit,
		SESRateBurst:  config.SESRateBurst,
		SESDailyQuota: config.SESDailyQuota,
		WorkerCount:   config.WorkerCount,
		BatchSize:     config.BatchSize,
		AWSRegion:     config.AWSRegion,
		HasOwnCreds:   config.HasOwnAWSCredentials(),
		FromEmail:     config.FromEmail,
		FromName:      config.FromName,
		ReplyTo:       config.ReplyTo,
	}, nil
}
