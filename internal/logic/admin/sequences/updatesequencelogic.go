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

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSequenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSequenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSequenceLogic {
	return &UpdateSequenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSequenceLogic) UpdateSequence(req *types.UpdateSequenceRequest) (resp *types.SequenceInfo, err error) {
	orgID, err := access.RequireOwner(l.ctx, l.svcCtx.DB, access.Sequence, req.Id, access.Member)
	if err != nil {
		return nil, err
	}

	sequence, err := l.svcCtx.DB.GetSequenceByID(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to get sequence %s: %v", req.Id, err)
		return nil, err
	}

	name := sequence.Name
	if req.Name != "" {
		name = req.Name
	}

	triggerEvent := sequence.TriggerEvent
	if req.TriggerEvent != "" {
		triggerEvent = req.TriggerEvent
	}

	sequenceType := sql.NullString{String: "lifecycle", Valid: true}
	if sequence.SequenceType.Valid {
		sequenceType = sequence.SequenceType
	}
	if req.SequenceType != "" {
		sequenceType = sql.NullString{String: req.SequenceType, Valid: true}
	}

	var sendHour sql.NullInt64
	if req.SendHour != nil {
		sendHour = sql.NullInt64{Int64: int64(*req.SendHour), Valid: true}
	}

	sendTimezone := sequence.SendTimezone
	if req.SendTimezone != "" {
		sendTimezone = sql.NullString{String: req.SendTimezone, Valid: true}
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	// Handle on_completion_sequence_id - use existing value if not provided in request
	onCompletionSequenceId := sequence.OnCompletionSequenceID
	if req.OnCompletionSequenceId != nil {
		if *req.OnCompletionSequenceId == "" {
			// Clear the chained sequence
			onCompletionSequenceId = sql.NullString{Valid: false}
		} else {
			if err := access.SameBrand(l.ctx, l.svcCtx.DB, access.Sequence, *req.OnCompletionSequenceId, orgID); err != nil {
				return nil, err
			}
			onCompletionSequenceId = sql.NullString{String: *req.OnCompletionSequenceId, Valid: true}
		}
	}

	// Handle list_id - use existing value if not provided in request
	listID := sequence.ListID
	if req.ListId != nil && *req.ListId != "" {
		if err := access.SameBrand(l.ctx, l.svcCtx.DB, access.List, *req.ListId, orgID); err != nil {
			return nil, err
		}
		newListID, parseErr := strconv.ParseInt(*req.ListId, 10, 64)
		if parseErr == nil {
			listID = sql.NullInt64{Int64: newListID, Valid: true}
		}
	}

	err = l.svcCtx.DB.UpdateSequence(l.ctx, db.UpdateSequenceParams{
		ID:                     req.Id,
		Name:                   name,
		TriggerEvent:           triggerEvent,
		IsActive:               isActive,
		SendHour:               sendHour,
		SendTimezone:           sendTimezone,
		SequenceType:           sequenceType,
		OnCompletionSequenceID: onCompletionSequenceId,
		ListID:                 listID,
	})
	if err != nil {
		l.Errorf("Failed to update sequence %s: %v", req.Id, err)
		return nil, err
	}

	var listIDStr string
	var listSlug, listName string
	if listID.Valid {
		listIDStr = strconv.FormatInt(listID.Int64, 10)
		list, err := l.svcCtx.DB.GetEmailList(l.ctx, listID.Int64)
		if err == nil {
			listSlug = list.Slug
			listName = list.Name
		}
	}

	var respSendHour *int
	if sendHour.Valid {
		h := int(sendHour.Int64)
		respSendHour = &h
	}

	respSendTimezone := "America/New_York"
	if sendTimezone.Valid && sendTimezone.String != "" {
		respSendTimezone = sendTimezone.String
	}

	respSequenceType := "lifecycle"
	if sequenceType.Valid && sequenceType.String != "" {
		respSequenceType = sequenceType.String
	}

	var respOnCompletionSequenceId, respOnCompletionSequenceName string
	if onCompletionSequenceId.Valid {
		respOnCompletionSequenceId = onCompletionSequenceId.String
		// Fetch the name of the completion sequence
		completionSequence, err := l.svcCtx.DB.GetSequenceByID(l.ctx, respOnCompletionSequenceId)
		if err == nil {
			respOnCompletionSequenceName = completionSequence.Name
		}
	}

	return &types.SequenceInfo{
		Id:                       sequence.ID,
		ListId:                   listIDStr,
		ListSlug:                 listSlug,
		ListName:                 listName,
		Slug:                     sequence.Slug,
		Name:                     name,
		TriggerEvent:             triggerEvent,
		SequenceType:             respSequenceType,
		IsActive:                 req.IsActive,
		SendHour:                 respSendHour,
		SendTimezone:             respSendTimezone,
		OnCompletionSequenceId:   respOnCompletionSequenceId,
		OnCompletionSequenceName: respOnCompletionSequenceName,
		CreatedAt:                utils.FormatNullString(sequence.CreatedAt),
	}, nil
}
