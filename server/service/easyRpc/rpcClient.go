package easyRpc

import (
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var pingTimeSecond = time.Second * 30
var pongTimeoutSecond = time.Second * 15
var connPool sync.Map // addr -> *grpc.ClientConn

func SetClientPingPongTime(ping time.Duration, pong time.Duration) {
	pingTimeSecond = ping
	pongTimeoutSecond = pong
}

func GetClientConn(addr string) (*grpc.ClientConn, error) {

	// fast path
	if v, ok := connPool.Load(addr); ok {
		return v.(*grpc.ClientConn), nil
	}

	conn, err := grpc.NewClient(
		addr,
		append([]grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultServiceConfig(`
				{
				  "loadBalancingConfig": [{"round_robin":{}}],
				  "retryPolicy": {
					"MaxAttempts": 5,
					"InitialBackoff": "0.2s",
					"MaxBackoff": "2s",
					"BackoffMultiplier": 2,
					"RetryableStatusCodes": ["UNAVAILABLE"]
				  }
				}
			`),

			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                pingTimeSecond,
				Timeout:             pongTimeoutSecond,
				PermitWithoutStream: true,
			}),

			grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(32<<20),
				grpc.MaxCallRecvMsgSize(32<<20),
			),
		})...,
	)
	if err != nil {
		return nil, err
	}

	// 双检入池
	actual, loaded := connPool.LoadOrStore(addr, conn)
	if loaded {
		_ = conn.Close()
		return actual.(*grpc.ClientConn), nil
	}

	return conn, nil
}

func CloseAll() {
	connPool.Range(func(_, v any) bool {
		_ = v.(*grpc.ClientConn).Close()
		return true
	})
}

func CloseConnect(addr string) {
	if conn, ok := connPool.LoadAndDelete(addr); ok {
		_ = conn.(*grpc.ClientConn).Close()
	}
}
