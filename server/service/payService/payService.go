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
			logger.InfoWithSprintf("[platform] begin to init google pay")
			googlePay, err := NewGooglePay(c.CredentialFile, c.PackageName)
			if err != nil {
				logger.ErrorBySprintf("[platform] Init google pay error:%v", err)
				panic("[platform] Init google pay error")
			}
			logger.InfoWithSprintf("[platform] google pay init success")
			m[PayGoogle] = googlePay
		}
	}
	logger.InfoWithSprintf("[platform] init payment service success")
	return &PaymentService{payChannels: m}
}

type PaymentService struct {
	payChannels map[PayChannel]PaymentGateway
}

func (p *PaymentService) ConsumeProduct(payChannel PayChannel, info *webProto.OrderInfo) error {
	pay := p.payChannels[payChannel]
	if pay == nil {
		logger.ErrorBySprintf("payment channel not supported productId:%s,token:%s,orderId:%s,err:%v", info.ProductId, info.Token, info.OrderId, PAY_CHANNEL_NOT_SUPPORTED)
		return PAY_CHANNEL_NOT_SUPPORTED
	}
	err := pay.Consume(info.ProductId, info.Token)
	if err != nil {
		logger.ErrorBySprintf("payment consume failed productId:%s,token:%s,orderId:%s,err:%v", info.ProductId, info.Token, info.OrderId, err)
		return err
	}
	return nil
}

func (p *PaymentService) Verify(payChannel PayChannel, info *webProto.OrderInfo) (*PayVerifyResult, error) {
	pay, ok := p.payChannels[payChannel]
	if !ok || pay == nil {
		logger.ErrorBySprintf("payment channel not supported productId:%s,token:%s,orderId:%s", info.ProductId, info.Token, info.OrderId)
		return nil, PAY_CHANNEL_NOT_SUPPORTED
	}
	result, err := pay.Verify(info.ProductId, info.Token)
	if err != nil {
		logger.ErrorBySprintf("payment verify failed productId:%s,token:%s,orderId:%s,err:%v", info.ProductId, info.Token, info.OrderId, err)
		return nil, NOT_PAID_ERROR
	}
	return result, nil
}
