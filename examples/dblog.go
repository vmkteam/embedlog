//go:build debug

package main

import (
	"context"
	"time"

	"github.com/vmkteam/embedlog"
)

type QueryLogger struct {
	embedlog.Logger
}

func (ql QueryLogger) BeforeQuery(ctx context.Context, event *pg.QueryEvent) (context.Context, error) {
	if event.Stash == nil {
		event.Stash = make(map[interface{}]interface{})
	}

	event.Stash["startedAt"] = time.Now()
	return ctx, nil
}

func (ql QueryLogger) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	method := fmt.Sprintf("%s.%s", zenrpc.NamespaceFromContext(ctx), zm.MethodFromContext(ctx))
	query, err := event.FormattedQuery()
	if err != nil {
		ql.Error(ctx, string(query), "err", err, "rpc", method)
	}

	var since time.Duration
	if event.Stash != nil {
		if v, ok := event.Stash["startedAt"]; ok {
			if startAt, ok := v.(time.Time); ok {
				since = time.Since(startAt)
			}
		}
	}

	ql.Log().DebugContext(ctx, string(query), "rpc", method, "duration", since)
	return nil
}

func NewQueryLogger(logger embedlog.Logger) QueryLogger {
	return QueryLogger{Logger: logger}
}
