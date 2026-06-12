package rpcController

import (
	"context"
	"time"

	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
)

func SendOperationToGateway(operation rpcPb.RPC_SERVER_OPERATION, param int64) {
	client, err := ServerNodeService.GetGatewayClient()
	if err != nil {
		logger.ErrorBySprintf("[rpc] send message to gateway error: %v", err)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := client.NotifyServerOperationHandler(ctx, &rpcPb.NotifyOperationMessage{
			Operation:      operation,
			OperationParam: param,
		})
		if err != nil {
			logger.ErrorBySprintf("SendOperationToGateway error: %v", err)
			return
		}
	}()
}
