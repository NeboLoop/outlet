package outbound

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOutboundRepliesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOutboundRepliesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOutboundRepliesLogic {
	return &ListOutboundRepliesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOutboundRepliesLogic) ListOutboundReplies(req *types.OutboundListRepliesRequest) (resp *types.OutboundReplyList, err error) {
	org, err := brand(l.ctx, l.svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	return ListOutboundReplies(l.ctx, l.svcCtx, org, req)
}

// ListOutboundReplies runs for a brand already resolved: the admin route's signed-in
// person, or the API key of /sdk/v1/outbound.
func ListOutboundReplies(ctx context.Context, svcCtx *svc.ServiceContext, org db.Organization, req *types.OutboundListRepliesRequest) (*types.OutboundReplyList, error) {
	rows, err := svcCtx.Outbound.ListReplies(ctx, org, req.Handled, req.Limit)
	if err != nil {
		return nil, err
	}
	out := &types.OutboundReplyList{Replies: make([]types.OutboundReplyInfo, 0, len(rows))}
	for _, r := range rows {
		out.Replies = append(out.Replies, types.OutboundReplyInfo{
			Id: r.ID, MailboxId: r.MailboxID, ContactId: r.ContactID.String, SequenceId: r.SequenceID.String, FromEmail: r.FromEmail,
			Subject: r.Subject, Body: r.Body, Classification: r.Classification, Handled: r.Handled == 1, ReceivedAt: r.ReceivedAt,
		})
	}
	return out, nil
}
