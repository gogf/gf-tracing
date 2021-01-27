package main

import (
	"context"
	"fmt"
	"gftracing/grpc+db+redis+log/protobuf/user"
	"github.com/gogf/gcache-adapter/adapter"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/katyusha/krpc"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"net"
	"time"
)

type server struct{}

const (
	JaegerEndpoint = "http://localhost:14268/api/traces"
	ServiceName    = "tracing-grpc-server"
)

func main() {
	flush := initTracer()
	defer flush()

	g.DB().GetCache().SetAdapter(adapter.NewRedis(g.Redis()))

	address := ":8000"
	listen, err := net.Listen("tcp", address)
	if err != nil {
		g.Log().Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			krpc.Server.UnaryError,
			krpc.Server.UnaryRecover,
			krpc.Server.UnaryTracing,
			krpc.Server.UnaryValidate,
		),
	)
	user.RegisterUserServer(s, &server{})
	g.Log().Printf("grpc server starts listening on %s", address)
	if err := s.Serve(listen); err != nil {
		g.Log().Fatalf("failed to serve: %v", err)
	}
}

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer() func() {
	// Create and install Jaeger export pipeline.
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint(JaegerEndpoint),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: ServiceName,
		}),
		jaeger.WithSDK(&trace.Config{DefaultSampler: trace.AlwaysSample()}),
	)
	if err != nil {
		g.Log().Fatal(err)
	}
	return flush
}

// Insert is a route handler for inserting user info into dtabase.
func (s *server) Insert(ctx context.Context, req *user.InsertReq) (*user.InsertRes, error) {
	res := user.InsertRes{}
	result, err := g.Table("user").Ctx(ctx).Insert(g.Map{
		"name": req.Name,
	})
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	res.Id = int32(id)
	return &res, nil
}

// Query is a route handler for querying user info. It firstly retrieves the info from redis,
// if there's nothing in the redis, it then does db select.
func (s *server) Query(ctx context.Context, req *user.QueryReq) (*user.QueryRes, error) {
	res := user.QueryRes{}
	err := g.Table("user").
		Ctx(ctx).
		Cache(5*time.Second, s.userCacheKey(req.Id)).
		WherePri(req.Id).
		Scan(&res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Delete is a route handler for deleting specified user info.
func (s *server) Delete(ctx context.Context, req *user.DeleteReq) (*user.DeleteRes, error) {
	res := user.DeleteRes{}
	_, err := g.Table("user").
		Ctx(ctx).
		Cache(-1, s.userCacheKey(req.Id)).
		WherePri(req.Id).
		Delete()
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *server) userCacheKey(id int32) string {
	return fmt.Sprintf(`userInfo:%d`, id)
}
