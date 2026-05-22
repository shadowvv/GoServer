package payService

type PayChannel string

const (
	PayGoogle PayChannel = "google"
	PayApple  PayChannel = "apple"
)

type PayVerifyResult struct {
	Success   bool
	OrderID   string
	IsSandBox bool
}

type PaymentGateway interface {
	Channel() PayChannel

	Verify(productID, token string) (*PayVerifyResult, error)

	Consume(productID, token string) error
}
