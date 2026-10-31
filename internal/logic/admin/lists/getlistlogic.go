package lists

import (
	"context"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetListLogic {
	return &GetListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetListLogic) GetList(req *types.GetListRequest) (resp *types.ListInfo, err error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid list ID: %w", err)
	}

	list, err := requireList(l.ctx, l.svcCtx, id, access.Member)
	if err != nil {
		return nil, err
	}

	subscriberCount, err := l.svcCtx.DB.CountListSubscribers(l.ctx, db.CountListSubscribersParams{
		ListID:       id,
		FilterStatus: "active",
	})
	if err != nil {
		l.Errorf("Failed to count subscribers for list %d: %v", id, err)
		return nil, err
	}

	return &types.ListInfo{
		Id:                     strconv.FormatInt(list.ID, 10),
		PublicId:               list.PublicID,
		OrgId:                  list.OrgID,
		Name:                   list.Name,
		Slug:                   list.Slug,
		Description:            list.Description.String,
		DoubleOptin:            list.DoubleOptin.Int64 == 1,
		ConfirmationSubject:    list.ConfirmationEmailSubject.String,
		ConfirmationBody:       list.ConfirmationEmailBody.String,
		SubscriberCount:        int(subscriberCount),
		CreatedAt:              utils.FormatNullString(list.CreatedAt),
		UpdatedAt:              utils.FormatNullString(list.UpdatedAt),
		ThankYouUrl:            list.ThankYouUrl.String,
		ConfirmRedirectUrl:     list.ConfirmRedirectUrl.String,
		AlreadySubscribedUrl:   list.AlreadySubscribedUrl.String,
		UnsubscribeRedirectUrl: list.UnsubscribeRedirectUrl.String,
		ThankYouEmailEnabled:   list.ThankYouEmailEnabled.Int64 == 1,
		ThankYouEmailSubject:   list.ThankYouEmailSubject.String,
		ThankYouEmailBody:      list.ThankYouEmailBody.String,
		GoodbyeEmailEnabled:    list.GoodbyeEmailEnabled.Int64 == 1,
		GoodbyeEmailSubject:    list.GoodbyeEmailSubject.String,
		GoodbyeEmailBody:       list.GoodbyeEmailBody.String,
		UnsubscribeBehavior:    list.UnsubscribeBehavior.String,
		UnsubscribeScope:       list.UnsubscribeScope.String,
	}, nil
}
