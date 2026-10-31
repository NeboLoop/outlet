package blocklist

import (
	"context"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSuppressedEmailsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSuppressedEmailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSuppressedEmailsLogic {
	return &ListSuppressedEmailsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSuppressedEmailsLogic) ListSuppressedEmails(req *types.ListSuppressedEmailsRequest) (resp *types.ListSuppressedEmailsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 50
	}
	offset := int64((page - 1) * limit)

	emails, err := l.svcCtx.DB.ListSuppressedEmails(l.ctx, db.ListSuppressedEmailsParams{
		OrgID:      orgID,
		PageOffset: offset,
		PageSize:   int64(limit),
	})
	if err != nil {
		l.Errorf("Failed to list suppressed emails: %v", err)
		return nil, err
	}

	total, err := l.svcCtx.DB.CountSuppressedEmails(l.ctx, orgID)
	if err != nil {
		l.Errorf("Failed to count suppressed emails: %v", err)
		return nil, err
	}

	result := make([]types.SuppressedEmailInfo, 0, len(emails))
	for _, email := range emails {
		blockAttempts := 0
		if email.BlockAttempts.Valid {
			blockAttempts = int(email.BlockAttempts.Int64)
		}
		createdAt := ""
		if email.CreatedAt.Valid {
			createdAt = email.CreatedAt.String
		}

		result = append(result, types.SuppressedEmailInfo{
			Id:            strconv.FormatInt(email.ID, 10),
			OrgId:         email.OrgID,
			Email:         email.Email,
			Reason:        email.Reason.String,
			Source:        email.Source.String,
			BlockAttempts: blockAttempts,
			CreatedAt:     createdAt,
		})
	}

	return &types.ListSuppressedEmailsResponse{
		Emails: result,
		Total:  int(total),
		Page:   page,
		Limit:  limit,
	}, nil
}
