package xf

import (
	"context"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func DialGRPC(host string, panicIfErrorOccurred bool) (*grpc.ClientConn, ErrorType) {
	// Set up a connection to the server.
	var err error
	backoffCfg := backoff.DefaultConfig
	backoffCfg.MaxDelay = 3 * time.Second // 最多间隔MaxDelay秒重新尝试连接

	// Discussion
	// With grpc.WithBlock() option set, grpc.Dial() will be blocked until connection be made.
	// Without grpc.WithBlock() option set, if connection cannot be made yet, Dial() returns a ClientConn object and no error anyway.
	// It seems Connection Backoff will handle retry connecting.
	conn, err := grpc.Dial(
		host,
		grpc.WithInsecure(),
		//grpc.WithBlock(),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 10, // 如果建立连接需要10秒，服务端或网络有问题。
			Backoff:           backoff.DefaultConfig,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithChainUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_zap.UnaryClientInterceptor(Logger, GRPCClientZapLogOption()),
			grpc_prometheus.UnaryClientInterceptor,
		)),
		grpc.WithChainStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_zap.StreamClientInterceptor(Logger, GRPCClientZapLogOption()),
			grpc_prometheus.StreamClientInterceptor,
		)),
	)

	if err != nil {
		et := ErrGRPCDialError(host, err)
		if panicIfErrorOccurred {
			panic(et)
		}
		//Error("Can't dial to grpc server %v. error=%v", c.host, err)
		return nil, et
	}
	Infof("Create connection to GRPC Server %s", host)

	return conn, nil
}

// GRPCClientZapLogOption almost the same compare to grpc_zap.DefaultMessageProducer. Additionally, log traceID.
func GRPCClientZapLogOption() grpc_zap.Option {
	return grpc_zap.WithMessageProducer(func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
		traceID := TraceIDFromOutgoing(ctx)

		//Infof("[%s] msg=%s; code=%v; duration=%vms; err=%v", traceID, msg, code, float32(duration.Integer/1000)/1000, err)
		ctxzap.Extract(ctx).Check(level, msg).Write(
			zap.Error(err),
			zap.String("grpc.code", code.String()),
			duration,
			zap.String("tid", traceID),
		)
	})
}

// GRPCServerZapLogOption almost the same compare to grpc_zap.DefaultMessageProducer. Additionally, log traceID.
func GRPCServerZapLogOption() grpc_zap.Option {
	return grpc_zap.WithMessageProducer(func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
		traceID := TraceIDFromIncoming(ctx)

		ctxzap.Extract(ctx).Check(level, msg).Write(
			zap.Error(err),
			zap.String("grpc.code", code.String()),
			duration,
			zap.String("tid", traceID),
		)
	})
}

// AddHeaderToGRPCRequest is an alias to metadata.AppendToOutgoingContext in case you don't know how to add header to GRPC request.
// Now you know, just call metadata.AppendToOutgoingContext.
func AddHeaderToGRPCRequest(context context.Context, kv ...string) context.Context {
	return metadata.AppendToOutgoingContext(context, kv...)
}
