package sequences

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type EnrollInSequenceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnrollInSequenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnrollInSequenceLogic {
	return &EnrollInSequenceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnrollInSequenceLogic) EnrollInSequence(req *types.EnrollSequenceRequest) (resp *types.EnrollSequenceResponse, err error) {
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok {
		return &types.EnrollSequenceResponse{Success: false}, fmt.Errorf("organization not found")
	}

	if req.Email == "" {
		return &types.EnrollSequenceResponse{Success: false}, fmt.Errorf("email is required")
	}
	if req.SequenceSlug == "" {
		return &types.EnrollSequenceResponse{Success: false}, fmt.Errorf("sequence_slug is required")
	}

	sequence, err := l.svcCtx.DB.GetSequenceByOrgAndSlug(l.ctx, db.GetSequenceByOrgAndSlugParams{
		OrgID: sql.NullString{String: orgID, Valid: true},
		Slug:  req.SequenceSlug,
	})
	if err != nil {
		l.Errorf("Sequence not found: %s, error: %v", req.SequenceSlug, err)
		return &types.EnrollSequenceResponse{Success: false}, fmt.Errorf("sequence not found")
	}

	if !sequence.IsActive.Valid || sequence.IsActive.Int64 != 1 {
		return &types.EnrollSequenceResponse{Success: false}, fmt.Errorf("sequence is not active")
	}

	contact, err := l.svcCtx.DB.GetContactByOrgAndEmail(l.ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: orgID, Valid: true},
		Email: req.Email,
	})
	if err != nil {
		l.Errorf("Contact not found: %s, error: %v", req.Email, err)
		return &types.EnrollSequenceResponse{Success: false}, fmt.Errorf("contact not found")
	}

	existingState, err := l.svcCtx.DB.GetContactSequenceState(l.ctx, db.GetContactSequenceStateParams{
		ContactID:  sql.NullString{String: contact.ID, Valid: true},
		SequenceID: sql.NullString{String: sequence.ID, Valid: true},
	})
	if err == nil {
		if existingState.IsActive.Valid && existingState.IsActive.Int64 == 1 {
			return &types.EnrollSequenceResponse{
				Success:      true,
				EnrollmentId: existingState.ID,
			}, nil
		}
		return &types.EnrollSequenceResponse{
			Success:      true,
			EnrollmentId: existingState.ID,
		}, nil
	}

	startPosition := int64(1)
	if req.StartAtStep > 0 {
		startPosition = int64(req.StartAtStep)
	}

	state, err := l.svcCtx.DB.CreateContactSequenceState(l.ctx, db.CreateContactSequenceStateParams{
		ID:              uuid.New().String(),
		ContactID:       sql.NullString{String: contact.ID, Valid: true},
		SequenceID:      sql.NullString{String: sequence.ID, Valid: true},
		CurrentPosition: sql.NullInt64{Int64: startPosition, Valid: true},
	})
	if err != nil {
		l.Errorf("Failed to create sequence state: %v", err)
		return &types.EnrollSequenceResponse{Success: false}, fmt.Errorf("failed to enroll in sequence")
	}

	template, err := l.svcCtx.DB.GetNextTemplate(l.ctx, db.GetNextTemplateParams{
		SequenceID: sql.NullString{String: sequence.ID, Valid: true},
		Position:   startPosition,
	})
	if err == nil {
		var scheduledFor time.Time

		if req.ScheduledFor != "" {
			parsedTime, parseErr := time.Parse(time.RFC3339, req.ScheduledFor)
			if parseErr == nil {
				scheduledFor = parsedTime
			} else {
				scheduledFor = time.Now().Add(time.Duration(template.DelayHours) * time.Hour)
			}
		} else {
			scheduledFor = time.Now().Add(time.Duration(template.DelayHours) * time.Hour)
		}

		trackingToken := uuid.New().String()

		_, queueErr := l.svcCtx.DB.QueueEmail(l.ctx, db.QueueEmailParams{
			ID:            uuid.New().String(),
			ContactID:     sql.NullString{String: contact.ID, Valid: true},
			TemplateID:    sql.NullString{String: template.ID, Valid: true},
			ScheduledFor:  scheduledFor.UTC().Format(time.RFC3339),
			TrackingToken: sql.NullString{String: trackingToken, Valid: true},
		})
		if queueErr != nil {
			l.Errorf("Failed to queue first email: %v", queueErr)
		}
	}

	l.Infof("Enrolled contact in sequence: org=%s email=%s sequence=%s enrollment_id=%s",
		orgID, req.Email, req.SequenceSlug, state.ID)

	return &types.EnrollSequenceResponse{
		Success:      true,
		EnrollmentId: state.ID,
	}, nil
}
