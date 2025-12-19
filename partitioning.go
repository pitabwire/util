package util

import (
	"context"
)

const ctxValueTenancyData = contextKeyType("tenancy_info")

type TenancyInfo interface {
	GetTenantID() string
	GetPartitionID() string
	GetProfileID() string
	GetAccessID() string
	GetContactID() string
	GetSessionID() string
	GetDeviceID() string
	GetRoles() []string
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
