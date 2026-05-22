package gameController

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/ServerNodeService"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/httpPlatform"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/logic/webProto"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/payService"
	"github.com/drop/GoServer/server/tool"
)

func RegisterRechargeMessage() {
	httpPlatform.RegisterHttpMessage("/consumeProduct", handleConsumeProduct)
	httpPlatform.RegisterHttpMessage("/gmConsumeProduct", handleGmConsumeProduct)
}

func handleGmConsumeProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req webProto.ConsumeProductReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	service := httpPlatform.GetPayService()
	for _, info := range req.OrderInfo {
		orderEntity, err := easyDB.GetServerEntityByWhere[model.RechargeOrderEntity](map[string]interface{}{"platform_order_id": info.OrderId})
		if err != nil {
			logger.ErrorBySprintf("[recharge] get order db error: %v,order:%v", err, info)
			sendErrorMessage(pb.ERROR_CODE_ORDER_NOT_FOUND, w)
			return
		}
		if orderEntity.Status != int32(enum.RECHARGE_ORDER_STATUS_CREATED) {
			logger.ErrorBySprintf("[recharge] recharge order status error: %v", req)
			sendErrorMessage(pb.ERROR_CODE_ORDER_ALREADY_CONSUME, w)
			return
		}

		err = service.ConsumeProduct(payService.PayGoogle, info)
		if err != nil {
			errorCode := pb.ERROR_CODE_SYSTEM_ERROR
			switch err {
			case payService.PAY_CHANNEL_NOT_SUPPORTED:
			case payService.NOT_PAID_ERROR:
				errorCode = pb.ERROR_CODE_ORDER_NOT_PAID
			}
			sendErrorMessage(errorCode, w)
			return
		}
		logger.InfoWithSprintf("[recharge] check order success: %v", info)
		orderEntity.Status = int32(enum.RECHARGE_ORDER_STATUS_PAYED)
		orderEntity.PayTime = tool.UnixNowMilli()
		err = easyDB.UpdateServerEntity[model.RechargeOrderEntity](orderEntity, map[string]interface{}{"status": orderEntity.Status, "pay_time": orderEntity.PayTime})
		if err != nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[recharge] save order db error: %v,order:%v", err, info))
			sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
			return
		}
		enum.PublishRecharge(dbService.RDB, orderEntity.UserId, orderEntity.Account, int64(orderEntity.Price))
		Success(orderEntity, w)
	}
}

func handleConsumeProduct(w http.ResponseWriter, r *http.Request) {
	logger.InfoWithSprintf("[recharge] begin handleConsumeProduct")

	if r.Method != http.MethodPost {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	var req webProto.ConsumeProductReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
		return
	}

	data, _ := json.Marshal(req)
	logger.InfoWithSprintf("[recharge] begin handleConsumeProduct req: %s", string(data))

	service := httpPlatform.GetPayService()
	for _, info := range req.OrderInfo {
		logger.InfoWithSprintf("[recharge] begin check order info:%+v", info)
		result, err := service.Verify(payService.PayGoogle, info)
		if err != nil {
			errorCode := pb.ERROR_CODE_SYSTEM_ERROR
			switch err {
			case payService.PAY_CHANNEL_NOT_SUPPORTED:
			case payService.NOT_PAID_ERROR:
				errorCode = pb.ERROR_CODE_ORDER_NOT_PAID
			}
			sendErrorMessage(errorCode, w)
			logger.ErrorBySprintf("[recharge] verify order error: %v,order:%v", err, info)
			return
		}
		realOrderId, err := strconv.ParseInt(result.OrderID, 10, 64)
		if err != nil {
			logger.ErrorBySprintf("[recharge] parse orderId error: %v,order:%v", err, info)
			sendErrorMessage(pb.ERROR_CODE_INVALID_REQUEST_PARAM, w)
			return
		}
		orderEntity, err := easyDB.GetServerEntityByWhere[model.RechargeOrderEntity](map[string]interface{}{"order_id": realOrderId})
		if err != nil {
			logger.ErrorBySprintf("[recharge] get order db error: %v,order:%v", err, info)
			sendErrorMessage(pb.ERROR_CODE_ORDER_NOT_FOUND, w)
			return
		}
		if orderEntity.UserId != req.PlayerId {
			logger.ErrorBySprintf("[recharge] recharge order userId error: reqPlayerId:%d,entityUserId:%d", req.PlayerId, orderEntity.UserId)
			continue
		}
		if orderEntity.Status != int32(enum.RECHARGE_ORDER_STATUS_CREATED) {
			logger.ErrorBySprintf("[recharge] recharge order status error: %v", req)
			sendErrorMessage(pb.ERROR_CODE_ORDER_ALREADY_CONSUME, w)
			return
		}
		err = service.ConsumeProduct(payService.PayGoogle, info)
		if err != nil {
			errorCode := pb.ERROR_CODE_SYSTEM_ERROR
			switch err {
			case payService.PAY_CHANNEL_NOT_SUPPORTED:
			case payService.NOT_PAID_ERROR:
				errorCode = pb.ERROR_CODE_ORDER_NOT_PAID
			}
			sendErrorMessage(errorCode, w)
			logger.ErrorBySprintf("[recharge] verify order error: %v,order:%v", err, info)
			return
		}
		logger.InfoWithSprintf("[recharge] check order success: %v", info)
		orderEntity.Status = int32(enum.RECHARGE_ORDER_STATUS_PAYED)
		orderEntity.PayTime = tool.UnixNowMilli()
		orderEntity.PlatformOrderId = info.OrderId
		orderEntity.PayToken = info.Token
		orderEntity.PayPlatform = "googleplay"
		orderEntity.IsSandBox = 0
		if result.IsSandBox {
			orderEntity.IsSandBox = 1
		}
		err = easyDB.UpdateServerEntity[model.RechargeOrderEntity](orderEntity, map[string]interface{}{"status": orderEntity.Status, "pay_time": orderEntity.PayTime, "platform_order_id": orderEntity.PlatformOrderId, "pay_token": orderEntity.PayToken, "is_sand_box": orderEntity.IsSandBox, "pay_platform": orderEntity.PayPlatform})
		if err != nil {
			logger.ErrorWithZapFields(fmt.Sprintf("[recharge] save order db error: %v,order:%v", err, info))
			sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
			return
		}
		Success(orderEntity, w)
	}
}

func Success(order *model.RechargeOrderEntity, w http.ResponseWriter) {
	logger.InfoWithSprintf("[recharge] begin send deliver message orderId:%d", order.OrderId)
	client, err := ServerNodeService.GetGatewayClient()
	if err != nil {
		logger.ErrorBySprintf("[recharge] get GetGatewayClient error: %v,orderId:%d,", err, order.OrderId)
		sendErrorMessage(pb.ERROR_CODE_SYSTEM_ERROR, w)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err = client.DeliverRechargeItem(ctx, &rpcPb.DeliverRechargeItemReq{
		OrderId:  order.OrderId,
		UserId:   order.UserId,
		Account:  order.Account,
		ServerId: order.ServerId,
	})
	if err != nil {
		logger.ErrorBySprintf("[recharge] send deliver message error: %v,orderId:%d,", err, order.OrderId)
		sendErrorMessage(pb.ERROR_CODE_SYSTEM_IS_BUSY, w)
		return
	}
	logger.InfoWithSprintf("[recharge] send deliver message end orderId:%d", order.OrderId)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&webProto.ConsumeProductResp{
		Code: int32(0),
	})
}
