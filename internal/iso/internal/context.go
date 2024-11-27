package internal

import "context"

type patchCtxKeyType string

const isoPatchCtxKey patchCtxKeyType = "iso-patch"

func WithPatch(ctx context.Context, patch []byte) context.Context {
	return context.WithValue(ctx, isoPatchCtxKey, patch)
}

func GetPatch(ctx context.Context) []byte {
	patch, ok := ctx.Value(isoPatchCtxKey).([]byte)
	if !ok {
		return nil
	}
	return patch
}
