package easyRpc

import (
	"net"

	"google.golang.org/grpc"
)

type ServerOptions struct {
	Address string
	Options []grpc.ServerOption
}

func NewServer(opt *ServerOptions, services ...func(*grpc.Server)) (*grpc.Server, error) {

	s := grpc.NewServer(
		opt.Options...,
	)

	for _, reg := range services {
		reg(s)
	}

	lis, err := net.Listen("tcp", opt.Address)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = s.Serve(lis)
	}()

	return s, nil
}

func StopServer(s *grpc.Server) {
	s.GracefulStop()
}
