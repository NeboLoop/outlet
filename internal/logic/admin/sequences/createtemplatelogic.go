package sequences

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTemplateLogic {
	return &CreateTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTemplateLogic) CreateTemplate(req *types.CreateTemplateRequest) (resp *types.TemplateInfo, err error) {
	if _, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.Sequence, req.SequenceId, access.Member); err != nil {
		return nil, err
	}

	templateType := sql.NullString{String: "simple", Valid: true}
	if req.TemplateType != "" {
		templateType = sql.NullString{String: req.TemplateType, Valid: true}
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	template, err := l.svcCtx.DB.CreateTemplate(l.ctx, db.CreateTemplateParams{
		ID:           uuid.New().String(),
		SequenceID:   sql.NullString{String: req.SequenceId, Valid: true},
		Position:     int64(req.Position),
		DelayHours:   int64(req.DelayHours),
		Subject:      req.Subject,
		HtmlBody:     req.HtmlBody,
		TemplateType: templateType,
		IsActive:     isActive,
	})
	if err != nil {
		return nil, err
	}

	return &types.TemplateInfo{
		Id:           template.ID,
		SequenceId:   template.SequenceID.String,
		Position:     int(template.Position),
		DelayHours:   int(template.DelayHours),
		Subject:      template.Subject,
		HtmlBody:     template.HtmlBody,
		TemplateType: template.TemplateType.String,
		IsActive:     template.IsActive.Int64 == 1,
		CreatedAt:    utils.FormatNullString(template.CreatedAt),
	}, nil
}
