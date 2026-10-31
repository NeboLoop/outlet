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

type CreateEntryRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateEntryRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateEntryRuleLogic {
	return &CreateEntryRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateEntryRuleLogic) CreateEntryRule(req *types.CreateEntryRuleRequest) (resp *types.EntryRuleResponse, err error) {
	if _, err := requireBrandOf(l.ctx, l.svcCtx, l.svcCtx.DB.GetSequenceOrgID, req.SequenceId, access.Member); err != nil {
		return nil, err
	}

	rule, err := l.svcCtx.DB.CreateEntryRule(l.ctx, db.CreateEntryRuleParams{
		ID:          uuid.New().String(),
		SequenceID:  req.SequenceId,
		TriggerType: req.TriggerType,
		SourceID:    req.SourceId,
		Priority:    sql.NullInt64{Int64: int64(req.Priority), Valid: true},
		IsActive:    sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to create entry rule: %v", err)
		return nil, err
	}

	var sourceName string
	if req.SourceId != "" {
		sourceIDInt, _ := strconv.ParseInt(req.SourceId, 10, 64)
		switch req.TriggerType {
		case "list_join":
			list, err := l.svcCtx.DB.GetEmailList(l.ctx, sourceIDInt)
			if err == nil {
				sourceName = list.Name
			}
		case "sequence_complete":
			seq, err := l.svcCtx.DB.GetSequenceByID(l.ctx, req.SourceId)
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
