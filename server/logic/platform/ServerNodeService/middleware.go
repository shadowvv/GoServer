package ServerNodeService

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"time"
)

func UnaryServerRecovery() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {

		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("internal server error")
				logger.ErrorWithZapFields("[platform][grpc] panic", zap.Error(err))
			}
		}()
		return handler(ctx, req)
	}
}

func UnaryServerLogging() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {

		start := time.Now()

		resp, err = handler(ctx, req)

		if err != nil {
			logger.ErrorBySprintf("[grpc] %s took=%v err=%v\n,code=%s", info.FullMethod, time.Since(start), err, status.Code(err))
		} else {
			logger.InfoWithSprintf("[grpc] %s took=%v err=%v\n", info.FullMethod, time.Since(start), err)
		}

		return
	}
}

func StreamServerLogging() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		start := time.Now()

		// 取对端地址
		var remoteAddr string
		if p, ok := peer.FromContext(ss.Context()); ok {
			remoteAddr = p.Addr.String()
		}

		logger.InfoWithSprintf("[grpc][stream] start method:%s remote:%s,isClientStream:%t,isServerStream:%t,", info.FullMethod, remoteAddr, info.IsClientStream, info.IsServerStream)

		// 执行业务 handler（阻塞直到 stream 结束）
		err := handler(srv, ss)

		cost := time.Since(start)
		st, _ := status.FromError(err)

		if err != nil {
			logger.ErrorBySprintf("[grpc][stream] end with error method=%s remote=%s code=%s cost=%s err=%s",
				info.FullMethod, remoteAddr, st.Code().String(), cost.String(), err.Error())
		} else {
			logger.InfoWithSprintf("[grpc][stream] end method=%s remote=%s code=%s", info.FullMethod, remoteAddr, cost.String())
		}

		return err
	}
}

func StreamServerRecovery() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorWithZapFields("[platform][grpc] panic", zap.Error(fmt.Errorf("internal server error")))
				return
			}
		}()
		return handler(srv, ss)
	}
}
