package main

import (
	"context"
	"time"

	"github.com/gogf/gf-tracing/tracing"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/net/ghttp"
	"github.com/gogf/gf/net/gtrace"
)

const (
	ServiceName       = "tracing-http-client"
	JaegerUdpEndpoint = "localhost:6831"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tp, err := tracing.InitJaeger(ServiceName, JaegerUdpEndpoint)
	if err != nil {
		g.Log().Ctx(ctx).Fatal(err)
	}

	// Cleanly shutdown and flush telemetry when the application exits.
	defer func(ctx context.Context) {
		// Do not make the application hang when it is shutdown.
		ctx, cancel = context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			g.Log().Ctx(ctx).Fatal(err)
		}
	}(ctx)

	StartRequests(ctx)
}

func StartRequests(ctx context.Context) {
	ctx, span := gtrace.NewSpan(ctx, "StartRequests")
	defer span.End()

	ctx = gtrace.SetBaggageValue(ctx, "name", "john")

	client := g.Client().Use(ghttp.MiddlewareClientTracing)

	content := client.Ctx(ctx).GetContent("http://127.0.0.1:8199/hello")
	g.Log().Ctx(ctx).Print(content)
}
