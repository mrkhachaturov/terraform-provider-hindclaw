package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nullableValue[T any] interface {
	IsSet() bool
	Get() *T
}

func nullableBoolToTF[N nullableValue[bool]](n N) types.Bool {
	if !n.IsSet() || n.Get() == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*n.Get())
}

func nullableStringToTF[N nullableValue[string]](n N) types.String {
	if !n.IsSet() || n.Get() == nil {
		return types.StringNull()
	}
	return types.StringValue(*n.Get())
}

func nullableInt32ToTF[N nullableValue[int32]](n N) types.Int64 {
	if !n.IsSet() || n.Get() == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*n.Get()))
}

func requiredNullableString[N nullableValue[string]](n N) (string, bool) {
	if !n.IsSet() || n.Get() == nil {
		return "", false
	}
	return *n.Get(), true
}

func stringSliceToTFListPreserveNullOnEmpty(ctx context.Context, prior types.List, values []string) (types.List, diag.Diagnostics) {
	if values == nil || (len(values) == 0 && prior.IsNull()) {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}
