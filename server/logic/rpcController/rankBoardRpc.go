package rpcController

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"io"
)

var rankCodec serviceInterface.CodecInterface
var rankRouter serviceInterface.RouterInterface

func InitRankProxy(codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) {
	rankCodec = codec
	rankRouter = router
}

func RegisterRankBoardRpcServer(s *grpc.Server) {
	rpcPb.RegisterRankServiceServer(s, &RankBoardRpcServer{})
}

type RankBoardRpcServer struct {
	rpcPb.UnimplementedRankServiceServer
}

func (r *RankBoardRpcServer) SayHello(ctx context.Context, req *rpcPb.HelloReq) (*rpcPb.HelloResp, error) {
	logger.InfoWithSprintf("[grpc] SayHello nodeInfo:%+v", req)
	return &rpcPb.HelloResp{}, nil
}

func (g *RankBoardRpcServer) ForwardRankBoardMessageHandler(stream grpc.BidiStreamingServer[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage]) error {

	ctx := stream.Context()
	rpcSender := NewGrpcSender[rpcPb.ForwardRankBoardMessage, rpcPb.BackwardRankBoardMessage](stream, 100)
	if rpcSender == nil {
		return fmt.Errorf("[grpc][stream] init rpc sender failed")
	}

	for {
		select {
		case <-ctx.Done():
			// gateway 主动断开 / 超时
			err := ctx.Err()
			st, _ := status.FromError(err)

			logger.ErrorBySprintf("[grpc][stream] ctx done code=%s err=%v", st.Code(), err)
			return err

		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				// 正常关闭
				logger.InfoWithSprintf("[net] rankBoard stream EOF")
				return nil
			}
			if err != nil {
				logger.ErrorWithZapFields(fmt.Sprintf("[net] recv failed: %v", err))
				return err
			}
			if msg == nil {
				continue
			}

			// ------------------ 路由 & 反序列化 ------------------
			rankBoardMessage := rankRouter.GetMessage(int32(msg.MsgId))
			if rankBoardMessage == nil {
				logger.ErrorBySprintf("[net] unknown msgId:%d userId:%d", msg.MsgId, msg.UserId)
				continue
			}

			if err = rankCodec.Unmarshal(msg.Payload, rankBoardMessage); err != nil {
				logger.ErrorBySprintf("[net] unmarshal failed userId:%d msgId:%d err:%v", msg.UserId, msg.MsgId, err)
				continue
			}
			rankRouter.Dispatch(logicSessionManager.NewRankBoardSession(rankCodec, msg.UserId, msg.RankId, msg.BackMessageId, msg.RespMsgId, rpcSender), int32(msg.MsgId), rankBoardMessage)
		}
	}
}
