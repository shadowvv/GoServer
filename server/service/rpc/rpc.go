package rpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func UnaryLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		cost := time.Since(start)
		if err != nil {
			logger.Error("RPC error", zap.String("method", info.FullMethod), zap.Error(err))
		} else {
			logger.Info("RPC call", zap.String("method", info.FullMethod), zap.Duration("cost", cost))
		}
		return resp, err
	}
}
