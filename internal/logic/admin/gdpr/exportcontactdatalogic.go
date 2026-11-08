package gdpr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/gdpr"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExportContactDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExportContactDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportContactDataLogic {
	return &ExportContactDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ExportContactData returns the contact's data in the response, for the app
// to save as a file: nothing is written on the server. It is gdpr.Export,
// the same export MCP gives.
func (l *ExportContactDataLogic) ExportContactData(req *types.GDPRExportRequest) (resp *types.GDPRExportResponse, err error) {
	contact, err := l.svcCtx.DB.GetContact(l.ctx, req.ContactId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errorx.NewNotFoundError("Contact not found")
	}
	if err != nil {
		return nil, err
	}
	if _, err := access.Require(l.ctx, l.svcCtx.DB, contact.OrgID.String, access.Member); err != nil {
		return nil, err
	}

	export, err := gdpr.Export(l.ctx, l.svcCtx.DB, contact.OrgID.String, contact.ID)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize export: %w", err)
	}
	return &types.GDPRExportResponse{
		Filename: fmt.Sprintf("gdpr-export-%s.json", contact.ID),
		Format:   "json",
		Data:     string(data),
	}, nil
}
