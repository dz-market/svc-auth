package slog

import (
	"context"
	"log/slog"
)

type attrsKey struct{}

func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing := attrsFrom(ctx)

	all := make([]slog.Attr, 0, len(existing)+len(attrs))
	all = append(all, existing...)
	all = append(all, attrs...)

	return context.WithValue(ctx, attrsKey{}, all)
}

func attrsFrom(ctx context.Context) []slog.Attr {
	attrs, ok := ctx.Value(attrsKey{}).([]slog.Attr)
	if !ok {
		return nil
	}

	return attrs
}

type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(attrsFrom(ctx)...)

	return h.Handler.Handle(ctx, r)
}
