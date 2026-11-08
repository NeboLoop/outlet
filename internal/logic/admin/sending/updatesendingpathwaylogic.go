package sending

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSendingPathwayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSendingPathwayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSendingPathwayLogic {
	return &UpdateSendingPathwayLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSendingPathwayLogic) UpdateSendingPathway(req *types.UpdateSendingPathwayRequest) (resp *types.SendingPathway, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Admin)
	if err != nil {
		return nil, err
	}
	p, err := l.svcCtx.EmailService.UpdatePathway(l.ctx, orgID, req.Id, email.PathwayInput{
		Name: req.Name, Kinds: req.Kinds,
		AWSRegion: req.AWSRegion, AWSAccessKey: req.AWSAccessKey, AWSSecretKey: req.AWSSecretKey,
		SMTPHost: req.SMTPHost, SMTPPort: req.SMTPPort, SMTPUsername: req.SMTPUsername, SMTPPassword: req.SMTPPassword,
		Limits: limitsIn(req.Limits),
	})
	if err != nil {
		return nil, err
	}
	out := pathwayOut(p)
	return &out, nil
}
