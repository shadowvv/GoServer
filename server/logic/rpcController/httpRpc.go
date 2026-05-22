package rpcController

import (
	"context"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"time"
)

func SendOperationToGateway(operation rpcPb.RPC_SERVER_OPERATION) {
	client, err := ServerNodeService.GetGatewayClient()
	if err != nil {
		logger.ErrorBySprintf("[rpc] send message to gateway error: %v", err)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := client.NotifyServerOperationHandler(ctx, &rpcPb.NotifyOperationMessage{
			Operation: operation,
		})
		if err != nil {
			logger.ErrorBySprintf("BroadcastOperationToGameNode error: %v", err)
			return
		}
	}()
}
