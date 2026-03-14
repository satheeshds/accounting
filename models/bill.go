package models

import "time"

// Bill represents a payable bill from a vendor.
type Bill struct {
	ID         int       `json:"id"`
	ContactID  *int      `json:"contact_id"`
	BillNumber string    `json:"bill_number"`
	IssueDate  *string   `json:"issue_date"`
	DueDate    *string   `json:"due_date"`
	Amount     Money     `json:"amount"`
	Status     string    `json:"status"`
	FileURL    *string   `json:"file_url"`
	Notes      *string   `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// Computed fields
	ContactName *string    `json:"contact_name,omitempty"`
	Allocated   Money      `json:"allocated"`   // sum of linked transaction_documents amounts
	Unallocated Money      `json:"unallocated"` // amount - allocated
	Items       []BillItem `json:"items"`
}

// BillInput is used for creating/updating bills.
type BillInput struct {
	ContactID  *int    `json:"contact_id"`
	BillNumber string  `json:"bill_number"`
	IssueDate  *string `json:"issue_date"`
	DueDate    *string `json:"due_date"`
	Amount     Money   `json:"amount"`
	Status     string  `json:"status"`
	FileURL    *string `json:"file_url"`
	Notes      *string `json:"notes"`
}

func (b *BillInput) Validate() string {
	if b.Amount < 0 {
		return "amount must be non-negative"
	}
	switch b.Status {
	case "", "draft", "partial", "received", "paid", "overdue", "cancelled":
	default:
		return "status must be one of: draft, partial, received, paid, overdue, cancelled"
	}
	if b.Status == "" {
		b.Status = "draft"
	}
	return ""
}

// BillItem represents a line item within a bill.
type BillItem struct {
	ID          int       `json:"id"`
	BillID      int       `json:"bill_id"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitPrice   Money     `json:"unit_price"`
	Amount      Money     `json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BillItemInput is used for creating/updating bill line items.
type BillItemInput struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   Money   `json:"unit_price"`
}

func (b *BillItemInput) Validate() string {
	if b.Description == "" {
		return "description is required"
	}
	if b.Quantity <= 0 {
		return "quantity must be positive"
	}
	if b.UnitPrice < 0 {
		return "unit_price must be non-negative"
	}
	return ""
}
