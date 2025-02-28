package paystack_local_payment_processor

import "time"

type PaystackGenertePaymentLinkResponse struct {
	Status  bool                           `json:"status"`
	Message string                         `json:"message"`
	Data    PaystackGenertePaymentLinkData `json:"data"`
}

type PaystackGenertePaymentLinkData struct {
	AuthURL *string `json:"authorization_url"`
}

type PaystackTransactionVerificationResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    TransactionData `json:"data"`
}

type TransactionData struct {
	ID                 int64         `json:"id"`
	Domain             string        `json:"domain"`
	Status             string        `json:"status"`
	Reference          string        `json:"reference"`
	ReceiptNumber      *string       `json:"receipt_number"`
	Amount             int64         `json:"amount"`
	Message            *string       `json:"message"`
	GatewayResponse    string        `json:"gateway_response"`
	PaidAt             time.Time     `json:"paid_at"`
	CreatedAt          time.Time     `json:"created_at"`
	Channel            string        `json:"channel"`
	Currency           string        `json:"currency"`
	IPAddress          string        `json:"ip_address"`
	Metadata           Metadata      `json:"metadata"`
	Log                Log           `json:"log"`
	Fees               int64         `json:"fees"`
	FeesSplit          interface{}   `json:"fees_split"`
	Authorization      Authorization `json:"authorization"`
	Customer           Customer      `json:"customer"`
	Plan               interface{}   `json:"plan"`
	Split              interface{}   `json:"split"`
	OrderID            *string       `json:"order_id"`
	PaidAtTime         time.Time     `json:"paidAt"`
	CreatedAtTime      time.Time     `json:"createdAt"`
	RequestedAmount    int64         `json:"requested_amount"`
	POSTransactionData interface{}   `json:"pos_transaction_data"`
	Source             interface{}   `json:"source"`
	FeesBreakdown      interface{}   `json:"fees_breakdown"`
	Connect            interface{}   `json:"connect"`
	TransactionDate    time.Time     `json:"transaction_date"`
	PlanObject         interface{}   `json:"plan_object"`
	Subaccount         interface{}   `json:"subaccount"`
}

type Metadata struct {
	PlanID      string `json:"planID"`
	WorkspaceID string `json:"workspaceID"`
	AppID       string `json:"appID"`
	Frequency   string `json:"frequency"`
	Reverse     string `json:"reverse"`
	AutoRenew   string `json:"autoRenew"`
}

type Log struct {
	StartTime int64     `json:"start_time"`
	TimeSpent int       `json:"time_spent"`
	Attempts  int       `json:"attempts"`
	Errors    int       `json:"errors"`
	Success   bool      `json:"success"`
	Mobile    bool      `json:"mobile"`
	Input     []string  `json:"input"`
	History   []History `json:"history"`
}

type History struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Time    int    `json:"time"`
}

type Authorization struct {
	AuthorizationCode         string  `json:"authorization_code"`
	Bin                       string  `json:"bin"`
	Last4                     string  `json:"last4"`
	ExpMonth                  string  `json:"exp_month"`
	ExpYear                   string  `json:"exp_year"`
	Channel                   string  `json:"channel"`
	CardType                  string  `json:"card_type"`
	Bank                      *string `json:"bank"`
	CountryCode               string  `json:"country_code"`
	Brand                     string  `json:"brand"`
	Reusable                  bool    `json:"reusable"`
	Signature                 *string `json:"signature"`
	AccountName               *string `json:"account_name"`
	ReceiverBankAccountNumber *string `json:"receiver_bank_account_number"`
	ReceiverBank              *string `json:"receiver_bank"`
	SenderBank                *string `json:"sender_bank"`
	SenderCountry             *string `json:"sender_country"`
	SenderBankAccountNumber   *string `json:"sender_bank_account_number"`
	SenderName                *string `json:"sender_name"`
	Narration                 *string `json:"narration"`
}

type Customer struct {
	ID                       int64   `json:"id"`
	FirstName                *string `json:"first_name"`
	LastName                 *string `json:"last_name"`
	Email                    string  `json:"email"`
	CustomerCode             string  `json:"customer_code"`
	Phone                    *string `json:"phone"`
	Metadata                 *string `json:"metadata"`
	RiskAction               string  `json:"risk_action"`
	InternationalFormatPhone *string `json:"international_format_phone"`
}
