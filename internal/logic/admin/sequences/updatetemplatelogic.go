package sequences

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTemplateLogic {
	return &UpdateTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTemplateLogic) UpdateTemplate(req *types.UpdateTemplateRequest) (resp *types.TemplateInfo, err error) {
	if _, err := requireBrandOf(l.ctx, l.svcCtx, l.svcCtx.DB.GetTemplateOrgID, req.Id, access.Member); err != nil {
		return nil, err
	}

	existing, err := l.svcCtx.DB.GetTemplateByID(l.ctx, req.Id)
	if err != nil {
		return nil, err
	}

	position := existing.Position
	if req.Position > 0 {
		position = int64(req.Position)
	}

	delayHours := existing.DelayHours
	if req.DelayHours > 0 {
		delayHours = int64(req.DelayHours)
	}

	subject := existing.Subject
	if req.Subject != "" {
		subject = req.Subject
	}

	htmlBody := existing.HtmlBody
	if req.HtmlBody != "" {
		htmlBody = req.HtmlBody
	}

	templateType := existing.TemplateType
	if req.TemplateType != "" {
		templateType = sql.NullString{String: req.TemplateType, Valid: true}
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	err = l.svcCtx.DB.UpdateTemplate(l.ctx, db.UpdateTemplateParams{
		ID:           req.Id,
		Position:     position,
		DelayHours:   delayHours,
		Subject:      subject,
		HtmlBody:     htmlBody,
		TemplateType: templateType,
		IsActive:     isActive,
	})
	if err != nil {
		return nil, err
	}

	template, err := l.svcCtx.DB.GetTemplateByID(l.ctx, req.Id)
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
