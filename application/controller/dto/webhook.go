package dto

import (
	"time"
)

type PaystackWebhookDTO struct {
	Event string     `json:"event"`
	Data  ChargeData `json:"data"`
}

type ChargeData struct {
	ID                 int64         `json:"id"`
	Domain             string        `json:"domain"`
	Status             string        `json:"status"`
	Reference          string        `json:"reference"`
	Amount             int64         `json:"amount"`
	Message            *string       `json:"message"`
	GatewayResponse    string        `json:"gateway_response"`
	PaidAt             time.Time     `json:"paid_at"`
	CreatedAt          time.Time     `json:"created_at"`
	Channel            string        `json:"channel"`
	Currency           string        `json:"currency"`
	IPAddress          string        `json:"ip_address"`
	Metadata           Metadata      `json:"metadata"`
	FeesBreakdown      *interface{}  `json:"fees_breakdown"`
	Log                *interface{}  `json:"log"`
	Fees               int64         `json:"fees"`
	FeesSplit          *interface{}  `json:"fees_split"`
	Authorization      Authorization `json:"authorization"`
	Customer           Customer      `json:"customer"`
	Plan               any           `json:"plan"`
	Subaccount         any           `json:"subaccount"`
	Split              any           `json:"split"`
	OrderID            *string       `json:"order_id"`
	RequestedAmount    int64         `json:"requested_amount"`
	PosTransactionData *interface{}  `json:"pos_transaction_data"`
	Source             Source        `json:"source"`
}

type Metadata struct {
	PlanID      string  `json:"planID"`
	WorkspaceID string  `json:"workspaceID"`
	AppID       *string `json:"appID"`
	Frequency   string  `json:"frequency"`
	Reverse     string  `json:"reverse"`
	AutoRenew   string  `json:"autoRenew"`
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
	SenderCountry             *string `json:"sender_country"`
	SenderBank                *string `json:"sender_bank"`
	SenderBankAccountNumber   *string `json:"sender_bank_account_number"`
	SenderName                *string `json:"sender_name"`
	Narration                 *string `json:"narration"`
	ReceiverBankAccountNumber *string `json:"receiver_bank_account_number"`
	ReceiverBank              *string `json:"receiver_bank"`
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

type Source struct {
	Type       string  `json:"type"`
	Source     string  `json:"source"`
	EntryPoint string  `json:"entry_point"`
	Identifier *string `json:"identifier"`
}
