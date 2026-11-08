package network

import (
	"context"
	"fmt"
	"time"

	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/signalnet"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NetworkLookupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNetworkLookupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NetworkLookupLogic {
	return &NetworkLookupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NetworkLookupLogic) NetworkLookup(req *types.NetworkLookupRequest) (resp *types.NetworkLookupResponse, err error) {
	if len(req.Hashes) == 0 || len(req.Hashes) > signalnet.MaxLookup {
		return nil, errorx.NewBadRequestError(fmt.Sprintf("hashes must contain 1 to %d items", signalnet.MaxLookup))
	}
	for _, h := range req.Hashes {
		if !signalnet.ValidHash(h) {
			return nil, errorx.NewBadRequestError("hashes must be 64 lowercase hex characters (SHA-256)")
		}
	}
	instanceID, _ := l.ctx.Value(middleware.NetworkInstanceIDKey).(string)
	signals, err := l.svcCtx.NetworkHub.Lookup(l.ctx, instanceID, req.Hashes)
	if err != nil {
		l.Errorf("network lookup failed: %v", err)
		return nil, errorx.NewInternalError("lookup failed")
	}
	resp = &types.NetworkLookupResponse{Signals: make([]types.NetworkSignal, 0, len(signals))}
	for _, s := range signals {
		resp.Signals = append(resp.Signals, types.NetworkSignal{
			Hash:          s.Hash,
			Senders:       s.Senders,
			Delivered:     s.Delivered,
			HardBounces:   s.HardBounces,
			Complaints:    s.Complaints,
			Engaged:       s.Engaged,
			LastEngagedAt: s.LastEngagedAt.Format(time.RFC3339), // zero time when never engaged
			TrapScore:     s.TrapScore,
			Toxic:         s.Toxic,
		})
	}
	return resp, nil
}
