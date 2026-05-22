package rpcController

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var socialCodec serviceInterface.CodecInterface
var socialRouter serviceInterface.RouterInterface

func InitSocialProxy(codec serviceInterface.CodecInterface, router serviceInterface.RouterInterface) {
	socialCodec = codec
	socialRouter = router
	gameCodec = codec
}

func RegisterSocialRpcServer(s *grpc.Server) {
	rpcPb.RegisterSocialServiceServer(s, &SocialRpcServer{})
}

func NotifyAllianceOperationToGateway(playerId int64, allianceId int64, oper pb.ALLIANCE_CHANGE_OPER) {
	client, err := ServerNodeService.GetGatewayClient()
	if err != nil {
		logger.ErrorBySprintf("[grpc] get gateway client error: %v,playerId:%d,allianceId:%d,oper:%d", err, playerId, allianceId, oper)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := client.NotifyAllianceChange(ctx, &rpcPb.NotifyAllianceChangeReq{
			UserId:     playerId,
			AllianceId: allianceId,
			ChangeOper: int32(oper),
		})
		if err != nil {
			logger.ErrorBySprintf("BroadcastOperationToGameNode error: %v", err)
			return
		}
	}()
}

type SocialRpcServer struct {
	rpcPb.UnimplementedSocialServiceServer
}

func (s *SocialRpcServer) SayHello(ctx context.Context, req *rpcPb.HelloReq) (*rpcPb.HelloResp, error) {
	logger.InfoWithSprintf("[grpc] social SayHello nodeInfo:%+v", req)
	return &rpcPb.HelloResp{}, nil
}

func (s *SocialRpcServer) ForwardSocialMessageHandler(stream grpc.BidiStreamingServer[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage]) error {
	ctx := stream.Context()
	rpcSender := NewGrpcSender[rpcPb.ForwardSocialMessage, rpcPb.BackwardSocialMessage](stream, 100)
	if rpcSender == nil {
		return fmt.Errorf("[grpc][stream] init social rpc sender failed")
	}

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			st, _ := status.FromError(err)
			logger.ErrorBySprintf("[grpc][stream] social ctx done code=%s err=%v", st.Code(), err)
			return err
		default:
			msg, err := stream.Recv()
			if err == io.EOF {
				logger.InfoWithSprintf("[net] social stream EOF")
				return nil
			}
			if err != nil {
				logger.ErrorWithZapFields(fmt.Sprintf("[net] social recv failed: %v", err))
				return err
			}
			if msg == nil {
				continue
			}

			socialMessage := socialRouter.GetMessage(int32(msg.MsgId))
			if socialMessage == nil {
				logger.ErrorBySprintf("[net] social unknown msgId:%d userId:%d", msg.MsgId, msg.UserId)
				continue
			}
			if err = socialCodec.Unmarshal(msg.Payload, socialMessage); err != nil {
				logger.ErrorBySprintf("[net] social unmarshal failed userId:%d msgId:%d err:%v", msg.UserId, msg.MsgId, err)
				continue
			}
			session := logicSessionManager.NewAllianceSession(socialCodec, msg.UserId, msg.AllianceId, msg.BackMessageId, rpcSender)
			socialRouter.Dispatch(session, int32(msg.MsgId), socialMessage)
		}
	}
}
