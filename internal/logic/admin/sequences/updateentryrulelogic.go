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

type UpdateEntryRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateEntryRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateEntryRuleLogic {
	return &UpdateEntryRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateEntryRuleLogic) UpdateEntryRule(req *types.UpdateEntryRuleRequest) (resp *types.EntryRuleResponse, err error) {
	if _, err := requireBrandOf(l.ctx, l.svcCtx, l.svcCtx.DB.GetEntryRuleOrgID, req.Id, access.Member); err != nil {
		return nil, err
	}

	var isActive sql.NullInt64
	if req.IsActive {
		isActive = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		isActive = sql.NullInt64{Int64: 0, Valid: true}
	}

	err = l.svcCtx.DB.UpdateEntryRule(l.ctx, db.UpdateEntryRuleParams{
		ID:       req.Id,
		Priority: sql.NullInt64{Int64: int64(req.Priority), Valid: true},
		IsActive: isActive,
	})
	if err != nil {
		l.Errorf("Failed to update entry rule: %v", err)
		return nil, err
	}

	rule, err := l.svcCtx.DB.GetEntryRule(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to get updated entry rule: %v", err)
		return nil, err
	}

	var sourceName string
	sourceID, _ := strconv.ParseInt(rule.SourceID, 10, 64)
	if sourceID > 0 {
		switch rule.TriggerType {
		case "list_join":
			list, err := l.svcCtx.DB.GetEmailList(l.ctx, sourceID)
			if err == nil {
				sourceName = list.Name
			}
		case "sequence_complete":
			seq, err := l.svcCtx.DB.GetSequenceByID(l.ctx, rule.SourceID)
			if err == nil {
				sourceName = seq.Name
			}
		}
	}

	return &types.EntryRuleResponse{
		EntryRule: types.EntryRuleInfo{
			Id:          rule.ID,
			SequenceId:  rule.SequenceID,
			TriggerType: rule.TriggerType,
			SourceId:    rule.SourceID,
			SourceName:  sourceName,
			Priority:    int(rule.Priority.Int64),
			IsActive:    rule.IsActive.Int64 == 1,
			CreatedAt:   utils.FormatNullString(rule.CreatedAt),
		},
	}, nil
}
