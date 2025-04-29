package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vmkteam/embedlog"
)

type Token string

// LogValue implements slog.LogValuer.
// It avoids revealing the token.
func (Token) LogValue() slog.Value {
	return slog.StringValue("REDACTED_TOKEN")
}

type ManagerMode string

const (
	ManagerModeV1 ManagerMode = "v1"
	ManagerModeV2 ManagerMode = "v2"
)

// Attr is a demo func that shows slog Attr feature.
func (m ManagerMode) Attr() slog.Attr {
	return slog.String("managerMode", string(m))
}

// MyManager is simple manager for testing purposes.
type MyManager struct {
	embedlog.Logger
}

// NewMyManager returns NewLogger manager.
func NewMyManager(logger embedlog.Logger) *MyManager {
	return &MyManager{
		Logger: logger,
	}
}

func (mm MyManager) Run(ctx context.Context) {
	mm.Print(ctx, "test run to stdout")
	mm.Error(ctx, "test run failed", "err", errors.New("const err"))
}

func (mm MyManager) ReportAllModes(ctx context.Context) {
	mm.Print(ctx, "modes", ManagerModeV1.Attr(), ManagerModeV2.Attr())
}

// Sample is a sample func with err checking.
func (mm MyManager) Sample(ctx context.Context) {
	id, rows := 1, 123
	lg := mm.With("id", id)

	var err error
	//nolint:gosec // tests
	if rand.IntN(2) == 1 {
		err = errors.New("random err")
		rows = 0
	}

	// here args will be included only if err == nil
	lg.PrintOrErr(ctx, "sample finished", err, "rows", rows)
	// if err != nil {
	//   lg.Error(ctx, "sample failed", "err", err)
	// } else {
	//   lg.Print(ctx, "sample succeeded", "rows", rows)
	// }
}

var (
	flVerbose = flag.Bool("verbose", true, "print verbose output")
	flJSON    = flag.Bool("json", false, "print output as JSON")
	flDev     = flag.Bool("dev", false, "uses development mode")
)

func main() {
	flag.Parse()
	verbose, isJSON, ctx := *flVerbose, *flJSON, context.Background()

	l := embedlog.NewLogger(verbose, isJSON)
	if *flDev {
		l = embedlog.NewDevLogger()
	}
	slog.SetDefault(l.Log()) // set default logger

	// goroutine test
	go func() {
		l3 := embedlog.NewLogger(verbose, isJSON)
		l3.Print(context.Background(), "l3 test", "token", Token("Secret"))
	}()

	m := NewMyManager(l)
	m.Run(ctx)

	// sample
	m.Sample(ctx)

	// test default slog
	slog.Info("this is default logger", "time", time.Now())

	// use group
	l2 := l.With("verbose", verbose, "isJSON", isJSON, ManagerModeV2.Attr())
	m2 := NewMyManager(l2)
	m2.Run(ctx)

	// report all modes
	m.ReportAllModes(ctx)

	// check metrics
	http.Handle("/metrics", LoggerMiddleware(l)(promhttp.Handler()))
	l.Printf("check metrics url=%v test=%q", "http://localhost:2112/metrics", "legacy mode")

	// test With()
	l.With("p1", true).Print(ctx, "test l1 with", ManagerModeV1.Attr())

	// test WithGroup()
	l2.WithGroup("l2").Print(ctx, "test l2 group", "v2", true, slog.Bool("test", true))

	// check metrics
	//nolint:gosec // tests
	err := http.ListenAndServe(":2112", nil)
	l.Errorf("err=%v", err)
}

// LoggerMiddleware simple middleware for http logging.
func LoggerMiddleware(logger embedlog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Print(r.Context(), "HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start),
				"token", Token("Secret"),
			)
		})
	}
}
