package rpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// 定义测试请求和响应结构体（模拟protobuf生成的代码）
type TestRequest struct {
	Message string
}

type TestResponse struct {
	Reply string
}

// TestServiceServer 实现测试服务
type TestServiceServer struct {
	// 模拟protobuf的UnimplementedServer
}

// Echo 方法实现
func (s *TestServiceServer) Echo(ctx context.Context, req *TestRequest) (*TestResponse, error) {
	return &TestResponse{
		Reply: fmt.Sprintf("Echo: %s", req.Message),
	}, nil
}

// Mock gRPC服务注册函数
func registerTestService(server *grpc.Server, svc *TestServiceServer) {
	// 在真实场景中这里会是 protobuf 生成的注册代码
	// testpb.RegisterTestServiceServer(server, svc)
	// 这里我们只是模拟注册过程
	fmt.Println("Test service registered")
}

func TestFullRPCFlow(t *testing.T) {
	// 创建logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 1. 测试服务端
	t.Run("Server Test", func(t *testing.T) {
		// 创建RPC服务器
		server := NewRPCServer(logger, grpc.UnaryInterceptor(UnaryLoggingInterceptor(logger)))

		// 使用bufconn进行测试，避免网络连接
		lis := bufconn.Listen(1024 * 1024)

		// 启动服务器goroutine
		go func() {
			// 注册测试服务
			registerTestService(server.server, &TestServiceServer{})

			if err := server.server.Serve(lis); err != nil {
				// 正常关闭时不报告错误
				if err != grpc.ErrServerStopped {
					t.Errorf("Server exited with error: %v", err)
				}
			}
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		// 创建客户端连接
		conn, err := grpc.DialContext(context.Background(), "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}),
			grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial bufnet: %v", err)
		}
		defer conn.Close()

		// 测试调用（模拟客户端调用）
		// 在真实场景中这里会是 protobuf 生成的客户端代码
		// client := testpb.NewTestServiceClient(conn)
		// resp, err := client.Echo(context.Background(), &TestRequest{Message: "Hello"})

		// 手动测试服务端功能
		testServer := &TestServiceServer{}
		resp, err := testServer.Echo(context.Background(), &TestRequest{Message: "Hello"})
		if err != nil {
			t.Fatalf("Echo failed: %v", err)
		}

		if resp.Reply != "Echo: Hello" {
			t.Errorf("Expected 'Echo: Hello', got '%s'", resp.Reply)
		}

		// 停止服务器
		server.Stop()
	})

	// 2. 测试连接池
	t.Run("Connection Pool Test", func(t *testing.T) {
		// 提前创建 bufconn.Listener 并复用
		lis := bufconn.Listen(1024 * 1024)

		// 启动模拟服务端（可选）
		server := grpc.NewServer()
		registerTestService(server, &TestServiceServer{})
		go func() {
			if err := server.Serve(lis); err != nil && err != grpc.ErrServerStopped {
				t.Errorf("Server exited with error: %v", err)
			}
		}()
		defer server.Stop()

		// 创建连接池
		pool := NewConnPool(func(addr string) (*grpc.ClientConn, error) {
			return grpc.DialContext(context.Background(), "bufnet",
				grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return lis.Dial()
				}),
				grpc.WithInsecure())
		})

		// 测试地址更新（这里地址只是标识，实际使用 bufconn）
		addrs := []string{"bufnet"}
		pool.UpdateAddresses(addrs)

		// 测试连接选择
		conn1, err := pool.PickConn()
		if err != nil {
			t.Fatalf("Failed to pick connection: %v", err)
		}
		defer conn1.Close()

		conn2, err := pool.PickConn()
		if err != nil {
			t.Fatalf("Failed to pick connection: %v", err)
		}
		defer conn2.Close()

		// 测试轮询功能
		if conn1 == conn2 {
			// 在只有2个地址的情况下，第3次选择应该回到第一个
			conn3, err := pool.PickConn()
			if err != nil {
				t.Fatalf("Failed to pick connection: %v", err)
			}

			conn4, err := pool.PickConn()
			if err != nil {
				t.Fatalf("Failed to pick connection: %v", err)
			}

			if conn3 != conn1 || conn4 != conn2 {
				t.Error("Round-robin selection not working correctly")
			}
		}

		// 测试关闭连接池
		pool.Close()
	})

	// 3. 测试日志拦截器
	t.Run("Logging Interceptor Test", func(t *testing.T) {
		interceptor := UnaryLoggingInterceptor(logger)

		// 创建一个模拟的handler
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return &TestResponse{Reply: "success"}, nil
		}

		// 创建模拟的UnaryServerInfo
		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.TestService/Echo",
		}

		// 测试拦截器
		resp, err := interceptor(context.Background(), &TestRequest{Message: "test"}, info, handler)
		if err != nil {
			t.Errorf("Interceptor failed: %v", err)
		}

		if resp == nil {
			t.Error("Expected response, got nil")
		}
	})
}
