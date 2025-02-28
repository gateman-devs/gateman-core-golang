package payment_types

type PaymentProcessor interface {
	GeneratePaymentLink(email string, metadata map[string]any, amount uint32, channels []PaymentChannel) (*GeneratePaymentLinkResponse, error)
	VerifyTransaction(id string) (interface{}, error)
	ReverseTransaction(id string, reason string) (interface{}, error)
}

type GeneratePaymentLinkResponse struct {
	Link string
}

type PaymentChannel string

const Card PaymentChannel = "card"
const Bank PaymentChannel = "bank"
const Transfer PaymentChannel = "bank_transfer"
const MobileMoney PaymentChannel = "mobile_money"
const USSD PaymentChannel = "ussd"
const QR PaymentChannel = "qr"
const EFT PaymentChannel = "eft"
