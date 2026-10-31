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

type GetSequenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSequenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSequenceLogic {
	return &GetSequenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSequenceLogic) GetSequence(req *types.GetSequenceRequest) (resp *types.SequenceDetailResponse, err error) {
	if _, err := requireBrandOf(l.ctx, l.svcCtx, l.svcCtx.DB.GetSequenceOrgID, req.Id, access.Member); err != nil {
		return nil, err
	}

	sequence, err := l.svcCtx.DB.GetSequenceByID(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to get sequence %s: %v", req.Id, err)
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

	templates, err := l.svcCtx.DB.ListTemplatesBySequence(l.ctx, sql.NullString{String: req.Id, Valid: true})
	if err != nil {
		l.Errorf("Failed to list templates for sequence %s: %v", req.Id, err)
		return nil, err
	}

	templateInfos := make([]types.TemplateInfo, 0, len(templates))
	for _, t := range templates {
		templateInfos = append(templateInfos, types.TemplateInfo{
			Id:           t.ID,
			SequenceId:   t.SequenceID.String,
			Position:     int(t.Position),
			DelayHours:   int(t.DelayHours),
			Subject:      t.Subject,
			HtmlBody:     t.HtmlBody,
			TemplateType: t.TemplateType.String,
			IsActive:     t.IsActive.Int64 == 1,
			CreatedAt:    utils.FormatNullString(t.CreatedAt),
		})
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

	sequenceType := "lifecycle"
	if sequence.SequenceType.Valid && sequence.SequenceType.String != "" {
		sequenceType = sequence.SequenceType.String
	}

	var onCompletionSequenceId, onCompletionSequenceName string
	if sequence.OnCompletionSequenceID.Valid {
		onCompletionSequenceId = sequence.OnCompletionSequenceID.String
		// Fetch the name of the completion sequence
		completionSequence, err := l.svcCtx.DB.GetSequenceByID(l.ctx, onCompletionSequenceId)
		if err == nil {
			onCompletionSequenceName = completionSequence.Name
		}
	}

	rules, _ := l.svcCtx.DB.ListEntryRulesBySequence(l.ctx, req.Id)
	entryRules := make([]types.EntryRuleInfo, 0, len(rules))
	for _, r := range rules {
		sourceName := ""
		if r.ListName.Valid {
			sourceName = r.ListName.String
		} else if r.SourceSequenceName.Valid {
			sourceName = r.SourceSequenceName.String
		}
		entryRules = append(entryRules, types.EntryRuleInfo{
			Id:          r.ID,
			SequenceId:  r.SequenceID,
			TriggerType: r.TriggerType,
			SourceId:    r.SourceID,
			SourceName:  sourceName,
			Priority:    int(r.Priority.Int64),
			IsActive:    r.IsActive.Int64 == 1,
			CreatedAt:   utils.FormatNullString(r.CreatedAt),
		})
	}

	return &types.SequenceDetailResponse{
		Sequence: types.SequenceInfo{
			Id:                       sequence.ID,
			ListId:                   listIDStr,
			ListSlug:                 listSlug,
			ListName:                 listName,
			Slug:                     sequence.Slug,
			Name:                     sequence.Name,
			TriggerEvent:             sequence.TriggerEvent,
			SequenceType:             sequenceType,
			IsActive:                 sequence.IsActive.Int64 == 1,
			SendHour:                 sendHour,
			SendTimezone:             sendTimezone,
			OnCompletionSequenceId:   onCompletionSequenceId,
			OnCompletionSequenceName: onCompletionSequenceName,
			EntryRules:               entryRules,
			CreatedAt:                utils.FormatNullString(sequence.CreatedAt),
		},
		Templates: templateInfos,
	}, nil
}
