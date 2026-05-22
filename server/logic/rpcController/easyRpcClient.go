package rpcController

import (
	"errors"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"google.golang.org/grpc"
	"sync"
	"time"
)

type StreamClient[Req any, Resp any] struct {
	mu     sync.Mutex
	create func() (grpc.BidiStreamingClient[Req, Resp], error) // 用于重连
	stream grpc.BidiStreamingClient[Req, Resp]

	sendCh chan *Req
	done   chan struct{}

	onReceiveResp func(*Resp)
}

func NewStreamClient[Req any, Resp any](create func() (grpc.BidiStreamingClient[Req, Resp], error), sendBuf int32, onReceiveResp func(*Resp)) *StreamClient[Req, Resp] {
	c := &StreamClient[Req, Resp]{
		create:        create,
		sendCh:        make(chan *Req, sendBuf),
		done:          make(chan struct{}),
		onReceiveResp: onReceiveResp,
	}
	go c.run()
	return c
}

func (c *StreamClient[Req, Resp]) run() {
	for {
		// 创建 stream
		stream, err := c.create()
		if err != nil {
			logger.ErrorBySprintf("[rpc] stream create failed: %v, retry after 3s", err)
			select {
			case <-time.After(3 * time.Second):
				continue
			case <-c.done:
				return
			}
		}

		c.mu.Lock()
		c.stream = stream
		c.mu.Unlock()

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.sendLoop()
		}()
		go func() {
			defer wg.Done()
			c.receiveLoop()
		}()

		wg.Wait()
		// 等待 send/recv 循环退出（即断线）
		logger.ErrorBySprintf("[rpc] stream disconnected, retrying after 3s")
		select {
		case <-time.After(3 * time.Second):
			continue
		case <-c.done:
			return
		}
	}
}

func (c *StreamClient[Req, Resp]) sendLoop() {
	for {
		select {
		case msg := <-c.sendCh:
			c.mu.Lock()
			stream := c.stream
			c.mu.Unlock()
			if stream == nil {
				logger.ErrorBySprintf("[rpc] send failed, stream nil")
				time.Sleep(time.Second)
				//c.sendCh <- msg // 放回队列重试
				return
			}
			if err := stream.Send(msg); err != nil {
				logger.ErrorBySprintf("[rpc] send failed: %v", err)
				//c.sendCh <- msg // 放回队列重试
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *StreamClient[Req, Resp]) receiveLoop() {
	for {
		c.mu.Lock()
		stream := c.stream
		c.mu.Unlock()
		if stream == nil {
			return
		}
		msg, err := stream.Recv()
		if err != nil {
			logger.ErrorBySprintf("[rpc] receive failed: %v", err)
			return
		}
		if msg != nil && c.onReceiveResp != nil {
			c.onReceiveResp(msg)
		}
	}
}

func (c *StreamClient[Req, Resp]) Send(msg *Req) error {
	select {
	case c.sendCh <- msg:
		return nil
	default:
		return errors.New("stream send queue full")
	}
}

func (c *StreamClient[Req, Resp]) Close() {
	close(c.done)
}

type EasyRpcClient[Req any, Resp any] struct {
	nodeId       int32
	shardNum     int32
	createClient func(shard int32) (grpc.BidiStreamingClient[Req, Resp], error)

	clients []*StreamClient[Req, Resp]
}

func (c *EasyRpcClient[Req, Resp]) SendMessage(id int64, msg *Req) error {
	if c.shardNum == 0 {
		return errors.New("shard num is zero")
	}
	idx := int(id % int64(c.shardNum))
	return c.clients[idx].Send(msg)
}

func (c *EasyRpcClient[Req, Resp]) Close() {
	for _, sc := range c.clients {
		sc.Close()
	}
}

func NewGameNodeClient(nodeId int32, createClient func(shard int32) (grpc.BidiStreamingClient[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage], error), shardNum int32, bufferSize int32, onReceiveResp func(*rpcPb.BackwardClientMessage)) *EasyRpcClient[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage] {
	c := &EasyRpcClient[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage]{
		nodeId:       nodeId,
		shardNum:     shardNum,
		createClient: createClient,
		clients:      make([]*StreamClient[rpcPb.ForwardGameMessage, rpcPb.BackwardClientMessage], shardNum),
	}
	for i := int32(0); i < shardNum; i++ {
		c.clients[i] = NewStreamClient(createClientWrapper(createClient, i), bufferSize, onReceiveResp)
	}
	return c
}

func NewRankBoardNodeClient(createClient func(shard int32) (grpc.BidiStreamingClient[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage], error), shardNum int32, bufferSize int32, onReceiveResp func(message *rpcPb.BackwardRankBoardMessage)) *EasyRpcClient[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage] {
	c := &EasyRpcClient[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage]{
		nodeId:       1,
		shardNum:     shardNum,
		createClient: createClient,
		clients:      make([]*StreamClient[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage], shardNum),
	}
	for i := int32(0); i < shardNum; i++ {
		c.clients[i] = NewStreamClient(createClientWrapper(createClient, i), bufferSize, onReceiveResp)
	}
	return c
}

func NewSocialNodeClient(createClient func(shard int32) (grpc.BidiStreamingClient[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage], error), shardNum int32, bufferSize int32, onReceiveResp func(message *rpcPb.BackwardSocialMessage)) *EasyRpcClient[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage] {
	c := &EasyRpcClient[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage]{
		nodeId:       1,
		shardNum:     shardNum,
		createClient: createClient,
		clients:      make([]*StreamClient[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage], shardNum),
	}
	for i := int32(0); i < shardNum; i++ {
		c.clients[i] = NewStreamClient(createClientWrapper(createClient, i), bufferSize, onReceiveResp)
	}
	return c
}

func createClientWrapper[Req any, Resp any](create func(shard int32) (grpc.BidiStreamingClient[Req, Resp], error), shard int32) func() (grpc.BidiStreamingClient[Req, Resp], error) {
	return func() (grpc.BidiStreamingClient[Req, Resp], error) {
		return create(shard)
	}
}
