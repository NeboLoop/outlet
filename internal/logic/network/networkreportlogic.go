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

type NetworkReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNetworkReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NetworkReportLogic {
	return &NetworkReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NetworkReportLogic) NetworkReport(req *types.NetworkReportRequest) (resp *types.NetworkReportResponse, err error) {
	if len(req.Events) == 0 || len(req.Events) > signalnet.MaxReport {
		return nil, errorx.NewBadRequestError(fmt.Sprintf("events must contain 1 to %d items", signalnet.MaxReport))
	}
	events := make([]signalnet.Event, 0, len(req.Events))
	for _, e := range req.Events {
		if !signalnet.ValidHash(e.Hash) {
			return nil, errorx.NewBadRequestError("event hash must be 64 lowercase hex characters (SHA-256)")
		}
		at, _ := time.Parse(time.RFC3339, e.At) // unparseable = now, clamped by the hub
		events = append(events, signalnet.Event{Hash: e.Hash, Kind: e.Kind, At: at})
	}
	instanceID, _ := l.ctx.Value(middleware.NetworkInstanceIDKey).(string)
	accepted, err := l.svcCtx.NetworkHub.Report(l.ctx, instanceID, events)
	if err != nil {
		l.Errorf("network report failed: %v", err)
		return nil, errorx.NewInternalError("report failed")
	}
	return &types.NetworkReportResponse{Accepted: accepted}, nil
}
