package lists

import (
	"context"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCustomFieldLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCustomFieldLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCustomFieldLogic {
	return &DeleteCustomFieldLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteCustomFieldLogic) DeleteCustomField(req *types.DeleteCustomFieldRequest) (resp *types.Response, err error) {
	if _, err := requireCustomField(l.ctx, l.svcCtx, req.ListId, req.FieldId, access.Member); err != nil {
		return nil, err
	}

	err = l.svcCtx.DB.DeleteCustomField(l.ctx, req.FieldId)
	if err != nil {
		l.Errorf("Failed to delete custom field %s: %v", req.FieldId, err)
		return nil, err
	}

	return &types.Response{
		Success: true,
		Message: "Custom field deleted successfully",
	}, nil
}
