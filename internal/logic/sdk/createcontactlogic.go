package sdk

import (
	"context"
	"database/sql"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateContactLogic {
	return &CreateContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateContactLogic) CreateContact(req *types.ContactRequest) (resp *types.ContactResponse, err error) {
	// Get org ID from context (set by API key middleware)
	orgID, ok := l.ctx.Value(middleware.OrgIDKey).(string)
	if !ok {
		return nil, nil
	}

	// Validate required fields
	if req.Email == "" {
		return nil, nil
	}

	// Check for existing contact by email within this org
	existingContact, err := l.svcCtx.DB.GetContactByOrgAndEmail(l.ctx, db.GetContactByOrgAndEmailParams{
		OrgID: sql.NullString{String: orgID, Valid: true},
		Email: req.Email,
	})
	if err == nil {
		// Contact exists - return their info
		l.Infof("Contact already exists: %s (%s)", existingContact.ID, existingContact.Email)
		return &types.ContactResponse{
			Id:    existingContact.ID,
			Email: existingContact.Email,
			Name:  existingContact.Name,
		}, nil
	}

	// Create new contact
	contact, err := l.svcCtx.DB.CreateContact(l.ctx, db.CreateContactParams{
		ID:     uuid.New().String(),
		OrgID:  sql.NullString{String: orgID, Valid: true},
		Name:   req.Name,
		Email:  req.Email,
		Source: sql.NullString{String: req.Source, Valid: req.Source != ""},
		Status: "new",
	})
	if err != nil {
		l.Errorf("Failed to create contact: %v", err)
		return nil, err
	}

	// Add tags if provided
	for _, tag := range req.Tags {
		_, _ = l.svcCtx.DB.AddContactTag(l.ctx, db.AddContactTagParams{
			ContactID: sql.NullString{String: contact.ID, Valid: true},
			Tag:       tag,
		})
	}

	l.Infof("Created contact: org=%s id=%s email=%s", orgID, contact.ID, contact.Email)

	return &types.ContactResponse{
		Id:    contact.ID,
		Email: contact.Email,
		Name:  contact.Name,
	}, nil
}
