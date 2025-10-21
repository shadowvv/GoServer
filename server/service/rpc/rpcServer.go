package rpc

import (
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type RPCServer struct {
	server *grpc.Server
	logger *zap.Logger
}

func NewRPCServer(logger *zap.Logger, opts ...grpc.ServerOption) *RPCServer {
	return &RPCServer{
		server: grpc.NewServer(opts...),
		logger: logger,
	}
}

func (s *RPCServer) RegisterService(registerFunc func(*grpc.Server)) {
	registerFunc(s.server)
}

func (s *RPCServer) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.logger.Info("RPC server started", zap.String("addr", addr))
	return s.server.Serve(lis)
}

func (s *RPCServer) Stop() {
	s.logger.Info("RPC server stopping...")
	s.server.GracefulStop()
}
