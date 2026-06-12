package payService

import (
	"context"
	"errors"

	"github.com/drop/GoServer/server/service/logger"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

type GooglePay struct {
	client         *androidpublisher.Service
	packageName    string
	credentialFile string
}

func NewGooglePay(credentialFile, packageName string) (*GooglePay, error) {
	svc, err := androidpublisher.NewService(
		context.Background(),
		option.WithAuthCredentialsFile(option.ServiceAccount, credentialFile),
	)
	if err != nil {
		return nil, err
	}
	if svc == nil {
		logger.ErrorBySprintf("[recharge] NewGooglePay error")
	}
	client := &GooglePay{
		client:         svc,
		packageName:    packageName,
		credentialFile: credentialFile,
	}
	return client, nil
}

func (g *GooglePay) Verify(productID, token string) (*PayVerifyResult, error) {
	if g.client == nil {
		return nil, errors.New("google pay client is nil")
	}
	resp, err := g.client.Purchases.Products.Get(g.packageName, productID, token).Do()
	if err != nil {
		return nil, err
	}
	if resp.PurchaseState != 0 {
		return nil, errors.New("google purchase not completed")
	}
	isSandBox := false
	if resp.PurchaseType != nil {
		isSandBox = *resp.PurchaseType == 0
	}
	return &PayVerifyResult{
		Success:   true,
		IsSandBox: isSandBox,
		OrderID:   resp.ObfuscatedExternalAccountId,
	}, nil
}

func (g *GooglePay) Consume(productID, token string) error {
	return g.client.Purchases.Products.Consume(g.packageName, productID, token).Do()
}

func (g *GooglePay) Channel() PayChannel {
	return PayGoogle
}
