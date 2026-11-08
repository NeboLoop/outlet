package tracking

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/logic/public"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TrackConfirmLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTrackConfirmLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TrackConfirmLogic {
	return &TrackConfirmLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TrackConfirmLogic) TrackConfirm(req *types.TrackConfirmRequest) (resp *types.TrackConfirmResponse, err error) {
	// First, try list subscription confirmation
	subscriber, err := l.svcCtx.DB.ConfirmListSubscription(l.ctx, sql.NullString{String: req.Token, Valid: true})
	if err == nil {
		l.Infof("List subscription confirmed: subscriber_id=%s", subscriber.ID)
		return &types.TrackConfirmResponse{
			Success: true,
			Message: "Subscription confirmed",
		}, nil
	}

	// Fall back to existing contact email confirmation logic
	confirmLogic := public.NewConfirmEmailLogic(l.ctx, l.svcCtx)
	confirmResp, err := confirmLogic.ConfirmEmail(&types.ConfirmEmailRequest{Token: req.Token})
	if err != nil {
		return &types.TrackConfirmResponse{Success: false, Message: "confirmation failed"}, nil
	}

	return &types.TrackConfirmResponse{
		Success:     confirmResp.Success,
		Message:     confirmResp.Message,
		RedirectUrl: confirmResp.Redirect,
	}, nil
}
