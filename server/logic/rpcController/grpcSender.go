package rpcController

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"sync/atomic"

	"github.com/drop/GoServer/server/service/logger"
	"google.golang.org/grpc"
)

var grpcSenderIdGen = atomic.Int32{}

type GrpcSender[Req any, Resp any] struct {
	id     int32
	stream grpc.BidiStreamingServer[Req, Resp]

	sendCh  chan *Resp
	closeCh chan struct{}
	closed  atomic.Bool
}

var _ logicCommon.GrpcSenderInterface[any, any] = (*GrpcSender[any, any])(nil)

func NewGrpcSender[Req any, Resp any](stream grpc.BidiStreamingServer[Req, Resp], buffer int) *GrpcSender[Req, Resp] {
	s := &GrpcSender[Req, Resp]{
		id:      grpcSenderIdGen.Add(1),
		stream:  stream,
		sendCh:  make(chan *Resp, buffer),
		closeCh: make(chan struct{}),
	}
	go s.sendLoop()
	return s
}

func (s *GrpcSender[Req, Resp]) sendLoop() {
	for {
		select {
		case msg := <-s.sendCh:
			if err := s.stream.Send(msg); err != nil {
				logger.ErrorWithZapFields(fmt.Sprintf("[grpc] send failed err:%v", err))
				s.Close()
				return
			}
		case <-s.closeCh:
			return
		}
	}
}

func (s *GrpcSender[Req, Resp]) Send(msg *Resp) error {
	if s.closed.Load() {
		return errors.New("grpc sender closed")
	}

	select {
	case s.sendCh <- msg:
		return nil
	default:
		return errors.New("grpc send queue full")
	}
}

func (s *GrpcSender[Req, Resp]) Close() {
	if s.closed.CompareAndSwap(false, true) {
		close(s.closeCh)
	}
}

func (s *GrpcSender[Req, Resp]) GetID() int32 {
	return s.id
}
