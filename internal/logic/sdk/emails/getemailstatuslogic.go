package emails

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEmailStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetEmailStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEmailStatusLogic {
	return &GetEmailStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEmailStatusLogic) GetEmailStatus(req *types.GetEmailStatusRequest) (resp *types.EmailStatusResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok {
		return nil, nil
	}

	if req.MessageId == "" {
		return nil, nil
	}

	send, err := l.svcCtx.DB.GetTransactionalSendByTrackingAndOrg(l.ctx, db.GetTransactionalSendByTrackingAndOrgParams{
		TrackingToken: sql.NullString{String: req.MessageId, Valid: true},
		OrgID:         orgID,
	})
	if err != nil {
		l.Errorf("Failed to get email status: %v", err)
		return nil, err
	}

	status := "queued"
	if send.Status.Valid {
		status = send.Status.String
	}

	opens := 0
	if send.OpenedAt.Valid {
		opens = 1
	}
	clicks := 0
	if send.ClickedAt.Valid {
		clicks = 1
	}

	resp = &types.EmailStatusResponse{
		MessageId: req.MessageId,
		To:        send.ToEmail,
		Subject:   send.SentSubject,
		Status:    status,
		Opens:     opens,
		Clicks:    clicks,
	}

	if send.SentAt.Valid {
		resp.SentAt = send.SentAt.String
	}
	if send.DeliveredAt.Valid {
		resp.DeliveredAt = send.DeliveredAt.String
	}
	if send.OpenedAt.Valid {
		resp.OpenedAt = send.OpenedAt.String
	}
	if send.ClickedAt.Valid {
		resp.ClickedAt = send.ClickedAt.String
	}

	if status == "bounced" && send.ErrorMessage.Valid {
		resp.BouncedAt = send.CreatedAt.String
		resp.BounceType = send.ErrorMessage.String
	}

	l.Infof("GetEmailStatus: org=%s messageId=%s status=%s", orgID, req.MessageId, status)

	return resp, nil
}
