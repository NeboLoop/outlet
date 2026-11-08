package lists

import (
	"context"
	"errors"
	"net/mail"
	"strconv"
	"strings"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"

	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddListSubscriberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddListSubscriberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddListSubscriberLogic {
	return &AddListSubscriberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// AddListSubscriber adds one person to a list, active at once, the way a CSV
// import does.
func (l *AddListSubscriberLogic) AddListSubscriber(req *types.AddListSubscriberRequest) (resp *types.Response, err error) {
	listID, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, errorx.NewBadRequestError("Invalid list ID.")
	}
	orgID, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.List, req.Id, access.Member)
	if err != nil {
		return nil, err
	}
	address := strings.TrimSpace(req.Email)
	if a, err := mail.ParseAddress(address); err != nil || a.Address != address {
		return nil, errorx.NewBadRequestError("Enter a valid email address.")
	}

	added, err := l.svcCtx.DB.AddSubscriber(l.ctx, orgID, listID, address, req.Name, "manual")
	if errors.Is(err, db.ErrBlockedDomain) {
		return nil, errorx.NewBadRequestError("This address's domain is on the brand's blocklist.")
	}
	if err != nil {
		l.Errorf("Failed to add subscriber: %v", err)
		return nil, errorx.NewInternalError("Couldn't add the subscriber.")
	}
	if !added {
		return &types.Response{Success: true, Message: "Already on this list"}, nil
	}
	return &types.Response{Success: true, Message: "Subscriber added"}, nil
}
