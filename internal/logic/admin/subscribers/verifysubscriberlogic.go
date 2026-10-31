package subscribers

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifySubscriberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVerifySubscriberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifySubscriberLogic {
	return &VerifySubscriberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerifySubscriberLogic) VerifySubscriber(req *types.SubscriberActionRequest) (resp *types.SubscriberInfo, err error) {
	contactID := req.Id
	if _, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.Contact, contactID, access.Member); err != nil {
		return nil, err
	}

	// Manually verify the contact
	err = l.svcCtx.DB.ManuallyVerifyContact(l.ctx, contactID)
	if err != nil {
		return nil, err
	}

	// Get updated contact
	contact, err := l.svcCtx.DB.GetContactByID(l.ctx, contactID)
	if err != nil {
		return nil, err
	}

	return contactToSubscriberInfo(contact), nil
}
