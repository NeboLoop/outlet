package transactional

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/logic/sdk/emails"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendTransactionalTestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Send a test transactional email as the signed-in member, through the same send as the SDK route
func NewSendTransactionalTestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendTransactionalTestLogic {
	return &SendTransactionalTestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SendTransactionalTest sends through emails.Send, the SDK route's send, so a
// test proves the template, sender and suppression rules the brand's apps
// get, without the brand's API key in the browser. A send that fails is an
// error here, for the page to show.
func (l *SendTransactionalTestLogic) SendTransactionalTest(req *types.SendEmailRequest) (resp *types.SendEmailResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}
	org, err := l.svcCtx.DB.GetOrganizationByID(l.ctx, orgID)
	if err != nil {
		return nil, err
	}
	// The response's message says why a send failed, for the page to show.
	resp, _ = emails.Send(l.ctx, l.svcCtx, org, req, emails.ViaAPI)
	if !resp.Success {
		return nil, errorx.NewBadRequestError(resp.Message)
	}
	return resp, nil
}
