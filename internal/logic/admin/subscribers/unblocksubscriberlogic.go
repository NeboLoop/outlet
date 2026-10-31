package subscribers

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnblockSubscriberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnblockSubscriberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnblockSubscriberLogic {
	return &UnblockSubscriberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnblockSubscriberLogic) UnblockSubscriber(req *types.SubscriberActionRequest) (resp *types.SubscriberInfo, err error) {
	contactID := req.Id
	if _, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.Contact, contactID, access.Member); err != nil {
		return nil, err
	}

	// Unblock the contact
	err = l.svcCtx.DB.UnblockContact(l.ctx, contactID)
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
