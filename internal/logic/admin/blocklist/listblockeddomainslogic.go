package blocklist

import (
	"context"
	"fmt"
	"strconv"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListBlockedDomainsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListBlockedDomainsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBlockedDomainsLogic {
	return &ListBlockedDomainsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListBlockedDomainsLogic) ListBlockedDomains(req *types.ListBlockedDomainsRequest) (resp *types.ListBlockedDomainsResponse, err error) {
	orgID, err := access.RequireBrand(l.ctx, l.svcCtx.DB, access.Member)
	if err != nil {
		return nil, err
	}

	offset := int64((req.Page - 1) * req.Limit)

	domains, err := l.svcCtx.DB.ListBlockedDomains(l.ctx, db.ListBlockedDomainsParams{
		OrgID:      orgID,
		PageOffset: offset,
		PageSize:   int64(req.Limit),
	})
	if err != nil {
		l.Errorf("Failed to list blocked domains: %v", err)
		return nil, fmt.Errorf("failed to list blocked domains: %w", err)
	}

	total, err := l.svcCtx.DB.CountBlockedDomains(l.ctx, orgID)
	if err != nil {
		l.Errorf("Failed to count blocked domains: %v", err)
		return nil, fmt.Errorf("failed to count blocked domains: %w", err)
	}

	domainInfos := make([]types.BlockedDomainInfo, 0, len(domains))
	for _, d := range domains {
		blockAttempts := 0
		if d.BlockAttempts.Valid {
			blockAttempts = int(d.BlockAttempts.Int64)
		}

		domainInfos = append(domainInfos, types.BlockedDomainInfo{
			Id:            strconv.FormatInt(d.ID, 10),
			OrgId:         d.OrgID,
			Domain:        d.Domain,
			Reason:        utils.FormatNullString(d.Reason),
			BlockAttempts: blockAttempts,
			CreatedAt:     utils.FormatNullString(d.CreatedAt),
			UpdatedAt:     utils.FormatNullString(d.UpdatedAt),
		})
	}

	return &types.ListBlockedDomainsResponse{
		Domains: domainInfos,
		Total:   int(total),
		Page:    req.Page,
		Limit:   req.Limit,
	}, nil
}
