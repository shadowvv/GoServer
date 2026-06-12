package payService

import (
	"errors"

	"github.com/drop/GoServer/server/logic/webProto"
	"github.com/drop/GoServer/server/service/logger"
)

type PayConfig struct {
	Platform       string `yaml:"platform"`
	CredentialFile string `yaml:"credentialFile"`
	PackageName    string `yaml:"packageName"`
}

var PAY_CHANNEL_NOT_SUPPORTED = errors.New("payment channel not supported")
var NOT_PAID_ERROR = errors.New("not paid")

func NewPaymentService(configs []*PayConfig) *PaymentService {
	m := make(map[PayChannel]PaymentGateway)
	for _, c := range configs {
		switch c.Platform {
		case "google":
			logger.InfoWithSprintf("[recharge] begin to init google pay")
			googlePay, err := NewGooglePay(c.CredentialFile, c.PackageName)
			if err != nil {
				logger.ErrorBySprintf("[recharge] Init google pay error:%v", err)
				panic("[recharge] Init google pay error")
			}
			logger.InfoWithSprintf("[recharge] google pay init success")
			m[PayGoogle] = googlePay
		}
	}
	logger.InfoWithSprintf("[recharge] init payment service success")
	return &PaymentService{payChannels: m}
}

type PaymentService struct {
	payChannels map[PayChannel]PaymentGateway
}

func (p *PaymentService) ConsumeProduct(payChannel PayChannel, info *webProto.OrderInfo) error {
	logger.InfoWithSprintf("[recharge] begin payment consume channel:%s,productId:%s,token:%s,orderId:%s", payChannel, info.ProductId, info.Token, info.OrderId)
	pay := p.payChannels[payChannel]
	if pay == nil {
		logger.ErrorBySprintf("[recharge] payment channel not supported productId:%s,token:%s,orderId:%s,err:%v", info.ProductId, info.Token, info.OrderId, PAY_CHANNEL_NOT_SUPPORTED)
		return PAY_CHANNEL_NOT_SUPPORTED
	}
	err := pay.Consume(info.ProductId, info.Token)
	if err != nil {
		logger.ErrorBySprintf("[recharge] payment consume failed productId:%s,token:%s,orderId:%s,err:%v", info.ProductId, info.Token, info.OrderId, err)
		return err
	}
	logger.InfoWithSprintf("[recharge] payment consume success channel:%s,productId:%s,token:%s,orderId:%s", payChannel, info.ProductId, info.Token, info.OrderId)
	return nil
}

func (p *PaymentService) Verify(payChannel PayChannel, info *webProto.OrderInfo) (*PayVerifyResult, error) {
	logger.InfoWithSprintf("[recharge] begin payment verify channel:%s,productId:%s,token:%s,orderId:%s", payChannel, info.ProductId, info.Token, info.OrderId)
	pay, ok := p.payChannels[payChannel]
	if !ok || pay == nil {
		logger.ErrorBySprintf("[recharge] payment channel not supported productId:%s,token:%s,orderId:%s", info.ProductId, info.Token, info.OrderId)
		return nil, PAY_CHANNEL_NOT_SUPPORTED
	}
	result, err := pay.Verify(info.ProductId, info.Token)
	if err != nil {
		logger.ErrorBySprintf("[recharge] payment verify failed productId:%s,token:%s,orderId:%s,err:%v", info.ProductId, info.Token, info.OrderId, err)
		return nil, NOT_PAID_ERROR
	}
	logger.InfoWithSprintf("[recharge] payment verify success channel:%s,productId:%s,token:%s,orderId:%s,resultOrderId:%s,isSandBox:%t", payChannel, info.ProductId, info.Token, info.OrderId, result.OrderID, result.IsSandBox)
	return result, nil
}
