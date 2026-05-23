package model

type OrderFilter struct {
	Keyword       string
	TradeNo       string
	Status        string
	PaymentMethod string
	StartTime     int64
	EndTime       int64
}
