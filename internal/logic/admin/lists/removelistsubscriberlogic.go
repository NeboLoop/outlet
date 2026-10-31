package lists

import (
	"context"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveListSubscriberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveListSubscriberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveListSubscriberLogic {
	return &RemoveListSubscriberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveListSubscriberLogic) RemoveListSubscriber(req *types.RemoveSubscriberRequest) (resp *types.Response, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid list ID: %w", err)
	}

	if _, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.List, strconv.FormatInt(listID, 10), access.Member); err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.RemoveSubscriberFromList(l.ctx, db.RemoveSubscriberFromListParams{
		ListID:    listID,
		ContactID: req.SubscriberId,
	})
	if err != nil {
		l.Errorf("Failed to remove subscriber: %v", err)
		return nil, fmt.Errorf("failed to remove subscriber: %w", err)
	}

	return &types.Response{
		Success: true,
		Message: "Subscriber removed successfully",
	}, nil
}
