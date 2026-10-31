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

type GetOrgEmailConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrgEmailConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrgEmailConfigLogic {
	return &GetOrgEmailConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrgEmailConfigLogic) GetOrgEmailConfig(req *types.GetOrgEmailConfigRequest) (resp *types.OrgEmailConfigInfo, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, req.OrgId, access.Admin); err != nil {
		return nil, err
	}

	config, err := email.GetOrgEmailConfig(l.ctx, l.svcCtx.DB, req.OrgId)
	if err != nil {
		return nil, fmt.Errorf("failed to get email config: %w", err)
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
