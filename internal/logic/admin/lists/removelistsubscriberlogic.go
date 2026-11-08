package lists

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
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

	// The subscriber ID is the list subscription's, as the list of
	// subscribers and the detail view give it.
	sub, err := l.svcCtx.DB.GetListSubscriberByID(l.ctx, req.SubscriberId)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && sub.ListID != listID) {
		return nil, errorx.NewNotFoundError("Subscriber not found on this list.")
	}
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.DB.RemoveSubscriberFromList(l.ctx, db.RemoveSubscriberFromListParams{
		ListID:    listID,
		ContactID: sub.ContactID,
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
