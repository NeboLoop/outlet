package sequences

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSequenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSequenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSequenceLogic {
	return &CreateSequenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSequenceLogic) CreateSequence(req *types.CreateSequenceRequest) (resp *types.SequenceInfo, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	sequenceType := req.SequenceType
	if sequenceType == "" {
		sequenceType = "lifecycle"
	}

	var listIDInt64 int64
	if req.ListId != "" {
		if err := requireSameBrand(l.ctx, listOrgID(l.svcCtx), req.ListId, orgID); err != nil {
			return nil, err
		}
		listIDInt64, _ = strconv.ParseInt(req.ListId, 10, 64)
	}

	sequence, err := l.svcCtx.DB.CreateSequence(l.ctx, db.CreateSequenceParams{
		ID:           uuid.New().String(),
		OrgID:        sql.NullString{String: orgID, Valid: true},
		ListID:       sql.NullInt64{Int64: listIDInt64, Valid: req.ListId != ""},
		Slug:         req.Slug,
		Name:         req.Name,
		TriggerEvent: req.TriggerEvent,
		IsActive:     sql.NullInt64{Int64: 1, Valid: true},
		SequenceType: sequenceType,
	})
	if err != nil {
		l.Errorf("Failed to create sequence: %v", err)
		return nil, err
	}

	var listIDStr string
	var listSlug, listName string
	if sequence.ListID.Valid {
		listIDStr = strconv.FormatInt(sequence.ListID.Int64, 10)
		list, err := l.svcCtx.DB.GetEmailList(l.ctx, sequence.ListID.Int64)
		if err == nil {
			listSlug = list.Slug
			listName = list.Name
		}
	}

	var sendHour *int
	if sequence.SendHour.Valid {
		h := int(sequence.SendHour.Int64)
		sendHour = &h
	}

	sendTimezone := "America/New_York"
	if sequence.SendTimezone.Valid && sequence.SendTimezone.String != "" {
		sendTimezone = sequence.SendTimezone.String
	}

	return &types.SequenceInfo{
		Id:           sequence.ID,
		ListId:       listIDStr,
		ListSlug:     listSlug,
		ListName:     listName,
		Slug:         sequence.Slug,
		Name:         sequence.Name,
		TriggerEvent: sequence.TriggerEvent,
		SequenceType: sequenceType,
		IsActive:     sequence.IsActive.Int64 == 1,
		SendHour:     sendHour,
		SendTimezone: sendTimezone,
		CreatedAt:    utils.FormatNullString(sequence.CreatedAt),
	}, nil
}
