package sequences

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSequencesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSequencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSequencesLogic {
	return &ListSequencesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSequencesLogic) ListSequences() (resp *types.SequenceListResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	rows, err := l.svcCtx.DB.ListSequencesByOrg(l.ctx, sql.NullString{String: orgID, Valid: true})
	if err != nil {
		l.Errorf("Failed to list sequences: %v", err)
		return nil, err
	}

	sequences := make([]types.SequenceInfo, 0, len(rows))
	for _, row := range rows {
		var listIDStr string
		var listSlug, listName string
		if row.ListID.Valid {
			listIDStr = strconv.FormatInt(row.ListID.Int64, 10)
		}
		if row.ListSlug.Valid {
			listSlug = row.ListSlug.String
		}
		if row.ListName.Valid {
			listName = row.ListName.String
		}

		var sendHour *int
		if row.SendHour.Valid {
			h := int(row.SendHour.Int64)
			sendHour = &h
		}

		sendTimezone := "America/New_York"
		if row.SendTimezone.Valid && row.SendTimezone.String != "" {
			sendTimezone = row.SendTimezone.String
		}

		sequenceType := "lifecycle"
		if row.SequenceType.Valid && row.SequenceType.String != "" {
			sequenceType = row.SequenceType.String
		}

		var onCompletionSequenceId, onCompletionSequenceName string
		if row.OnCompletionSequenceID.Valid {
			onCompletionSequenceId = row.OnCompletionSequenceID.String
		}
		if row.OnCompletionSequenceName.Valid {
			onCompletionSequenceName = row.OnCompletionSequenceName.String
		}

		sequences = append(sequences, types.SequenceInfo{
			Id:                       row.ID,
			ListId:                   listIDStr,
			ListSlug:                 listSlug,
			ListName:                 listName,
			Slug:                     row.Slug,
			Name:                     row.Name,
			TriggerEvent:             row.TriggerEvent,
			SequenceType:             sequenceType,
			IsActive:                 row.IsActive.Int64 == 1,
			SendHour:                 sendHour,
			SendTimezone:             sendTimezone,
			OnCompletionSequenceId:   onCompletionSequenceId,
			OnCompletionSequenceName: onCompletionSequenceName,
			CreatedAt:                utils.FormatNullString(row.CreatedAt),
		})
	}

	return &types.SequenceListResponse{
		Sequences: sequences,
	}, nil
}
