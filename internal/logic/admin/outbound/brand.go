package outbound

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
)

// The admin routes run the same outbound service as the MCP tools, for the
// brand named by X-Org-Id. Setting up sending (the switch, the DNS token,
// domains and mailboxes) needs the brand's admin; the rest, a member.

func brand(ctx context.Context, svcCtx *svc.ServiceContext, need access.Role) (db.Organization, error) {
	orgID, err := access.RequireBrand(ctx, svcCtx.DB, need)
	if err != nil {
		return db.Organization{}, err
	}
	return svcCtx.DB.GetOrganizationByID(ctx, orgID)
}

// apiErr turns the service's refusals into a 400 the app shows as is.
func apiErr(err error) error {
	var r *outbound.Refusal
	if errors.As(err, &r) {
		return errorx.NewBadRequestError(r.Msg)
	}
	return err
}

// convert copies a service value into its API type; both use the same JSON names.
func convert[T any](src any) (*T, error) {
	raw, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// convertInto reads an API request into the service's input type.
func convertInto[T any](src any) (T, error) {
	var out T
	raw, err := json.Marshal(src)
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return out, err
}

func ok(msg string) *types.Response { return &types.Response{Success: true, Message: msg} }

// saveSegment creates a segment (id empty) or changes one, from its request.
func saveSegment(ctx context.Context, svcCtx *svc.ServiceContext, id string, req any) (*types.OutboundSegment, error) {
	org, err := brand(ctx, svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	in, err := convertInto[outbound.SegmentInput](req)
	if err != nil {
		return nil, err
	}
	sg, err := svcCtx.Outbound.SaveSegment(ctx, org, id, in)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundSegment](sg)
}

// saveMaterial creates a material (id empty) or changes one, from its request.
func saveMaterial(ctx context.Context, svcCtx *svc.ServiceContext, id string, req any) (*types.OutboundMaterial, error) {
	org, err := brand(ctx, svcCtx, access.Member)
	if err != nil {
		return nil, err
	}
	in, err := convertInto[outbound.MaterialInput](req)
	if err != nil {
		return nil, err
	}
	m, err := svcCtx.Outbound.SaveMaterial(ctx, org, id, in)
	if err != nil {
		return nil, apiErr(err)
	}
	return convert[types.OutboundMaterial](m)
}
