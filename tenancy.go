package util

import (
	"context"
)

const ctxValueTenancyData = contextKeyType("tenancy_info")

type TenancyInfo interface {
	GetTenantID() string
	GetPartitionID() string
	GetAccessID() string
}

func SetTenancy(ctx context.Context, tenancyInfo TenancyInfo) context.Context {
	return context.WithValue(ctx, ctxValueTenancyData, tenancyInfo)
}

func GetTenancy(ctx context.Context) TenancyInfo {
	info, ok := ctx.Value(ctxValueTenancyData).(TenancyInfo)
	if !ok {
		return nil
	}
	return info
}
