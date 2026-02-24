package models

import "time"

// Invoice represents a receivable invoice to a customer.
type Invoice struct {
	ID            int       `json:"id"`
	ContactID     *int      `json:"contact_id"`
	InvoiceNumber string    `json:"invoice_number"`
	IssueDate     *string   `json:"issue_date"`
	DueDate       *string   `json:"due_date"`
	Amount        Money     `json:"amount"`
	Status        string    `json:"status"`
	FileURL       *string   `json:"file_url"`
	Notes         *string   `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Computed fields
	ContactName *string `json:"contact_name,omitempty"`
	Allocated   Money   `json:"allocated"`
	Unallocated Money   `json:"unallocated"`
}

// InvoiceInput is used for creating/updating invoices.
type InvoiceInput struct {
	ContactID     *int    `json:"contact_id"`
	InvoiceNumber string  `json:"invoice_number"`
	IssueDate     *string `json:"issue_date"`
	DueDate       *string `json:"due_date"`
	Amount        Money   `json:"amount"`
	Status        string  `json:"status"`
	FileURL       *string `json:"file_url"`
	Notes         *string `json:"notes"`
}

func (i *InvoiceInput) Validate() string {
	if i.Amount < 0 {
		return "amount must be non-negative"
	}
	switch i.Status {
	case "", "draft", "partial", "sent", "paid", "received", "overdue", "cancelled":
	default:
		return "status must be one of: draft, partial, sent, paid, received, overdue, cancelled"
	}
	if i.Status == "" {
		i.Status = "draft"
	}
	return ""
}
