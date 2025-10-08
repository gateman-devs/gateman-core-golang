package payments

import (
	"os"

	"gateman.io/infrastructure/network"
	paystack_local_payment_processor "gateman.io/infrastructure/payments/paystack"
	payment_types "gateman.io/infrastructure/payments/types"
)

var PaymentProcessor payment_types.PaymentProcessor

func InitialisePaymentProcessor() {
	PaymentProcessor = &paystack_local_payment_processor.PaystackPaymentProcessor{
		Network: &network.NetworkController{
			BaseUrl: os.Getenv("PAYSTACK_BASE_URL"),
		},
		AuthToken: os.Getenv("PAYSTACK_ACCESS_TOKEN"),
	}
}
