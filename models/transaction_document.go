package models

import "time"

// TransactionDocument links a transaction to a bill, invoice, payout or recurring payment with an allocated amount.
type TransactionDocument struct {
	ID            int       `json:"id"`
	TransactionID int       `json:"transaction_id"`
	DocumentType  string    `json:"document_type"` // bill, invoice, payout or recurring_payment
	DocumentID    int       `json:"document_id"`
	Amount        Money     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

// TransactionDocumentInput is used for linking transactions to bills, invoices, payouts or recurring payments.
type TransactionDocumentInput struct {
	DocumentType string `json:"document_type"`
	DocumentID   int    `json:"document_id"`
	Amount       Money  `json:"amount"`
}

func (td *TransactionDocumentInput) Validate() string {
	switch td.DocumentType {
	case "bill", "invoice", "payout", "recurring_payment":
	default:
		return "document_type must be one of: bill, invoice, payout or recurring_payment"
	}
	if td.DocumentID <= 0 {
		return "document_id is required"
	}
	if td.Amount <= 0 {
		return "amount must be positive"
	}
	return ""
}
