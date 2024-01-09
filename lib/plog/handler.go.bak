package plog

/*
 * handler.go
 * Log handler with a variable level and runtime resolution.
 * By J. Stuart McMurray
 * Created 20230818
 * Last Modified 20231006
 */

import (
	"context"
	"log/slog"
)

// Handler is identical to the default slog handler except that it allows for a
// settable level and that it waits until Handle is called to resolve
// slog.Attrs supplied by WithAttrs.  Handler's methods satisfy slog.Handler.
type Handler struct {
	inner slog.Handler
	lv    *slog.LevelVar
	attrs []slog.Attr
	group string
}

// NewHandler returns a new Handler wrapping h whose level is set by lv.
func NewHandler(lv *slog.LevelVar, h slog.Handler) Handler {
	return Handler{inner: h, lv: lv}
}

func (h Handler) Handle(ctx context.Context, rec slog.Record) error {
	/* If we have attrs to add, add them. */
	if 0 != len(h.attrs) {
		rec = rec.Clone()
		rec.AddAttrs(h.attrs...)
	}

	/* If we have a group, collect all of the attrs we know about into
	their own group and pass that to the inner handler. */
	if "" != h.group {
		/* Sliceify the attrs, so as to stick in a group. */
		attrs := make([]any, 0, rec.NumAttrs())
		rec.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a)
			return true
		})
		/* Roll a new record with the Group as the only attr. */
		nrec := slog.NewRecord(
			rec.Time,
			rec.Level,
			rec.Message,
			rec.PC,
		)
		nrec.AddAttrs(slog.Group(h.group, attrs...))
		/* Prep it to be passed to the inner handler. */
		rec = nrec
	}

	/* Pass the record, maybe a modified copy or something, to the inner
	handler. */
	return h.inner.Handle(ctx, rec)
}

func (h Handler) Enabled(ctx context.Context, level slog.Level) bool {
	if nil == h.lv {
		return h.inner.Enabled(ctx, level)
	}
	return h.lv.Level() <= level
}

func (h Handler) WithGroup(name string) slog.Handler {
	if "" == name {
		return h
	}
	return Handler{inner: h, group: name}
}

func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if 0 == len(attrs) {
		return h
	}
	return Handler{inner: h, attrs: attrs}
}
