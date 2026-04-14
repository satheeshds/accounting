package store

// DashboardData holds aggregate statistics for the dashboard.
type DashboardData struct {
	TotalAccounts     int `json:"total_accounts"`
	TotalContacts     int `json:"total_contacts"`
	TotalBills        int `json:"total_bills"`
	TotalInvoices     int `json:"total_invoices"`
	TotalPayouts      int `json:"total_payouts"`
	TotalTransactions int `json:"total_transactions"`

	BillsPayable       int `json:"bills_payable"`
	InvoicesReceivable int `json:"invoices_receivable"`
	PayoutsReceived    int `json:"payouts_received"`

	OverdueBills    int `json:"overdue_bills"`
	OverdueInvoices int `json:"overdue_invoices"`

	RecentTransactions []map[string]any `json:"recent_transactions"`
}

// GetDashboard retrieves aggregate dashboard statistics.
func (s *Store) GetDashboard() (DashboardData, error) {
	var d DashboardData

	if err := s.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&d.TotalAccounts); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM contacts").Scan(&d.TotalContacts); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM bills").Scan(&d.TotalBills); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM invoices").Scan(&d.TotalInvoices); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM payouts").Scan(&d.TotalPayouts); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&d.TotalTransactions); err != nil {
		return DashboardData{}, err
	}

	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount - (SELECT COALESCE(SUM(td.amount), 0) FROM transaction_documents td WHERE td.document_type = 'bill' AND td.document_id = bills.id)), 0) 
		FROM bills WHERE status NOT IN ('paid', 'cancelled')`).Scan(&d.BillsPayable); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount - (SELECT COALESCE(SUM(td.amount), 0) FROM transaction_documents td WHERE td.document_type = 'invoice' AND td.document_id = invoices.id)), 0) 
		FROM invoices WHERE status NOT IN ('paid', 'received', 'cancelled')`).Scan(&d.InvoicesReceivable); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow("SELECT COALESCE(SUM(final_payout_amt), 0) FROM payouts").Scan(&d.PayoutsReceived); err != nil {
		return DashboardData{}, err
	}

	if err := s.db.QueryRow("SELECT COUNT(*) FROM bills WHERE status = 'overdue'").Scan(&d.OverdueBills); err != nil {
		return DashboardData{}, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM invoices WHERE status = 'overdue'").Scan(&d.OverdueInvoices); err != nil {
		return DashboardData{}, err
	}

	rows, err := s.db.Query(`SELECT t.id, t.type, t.amount, t.transaction_date, t.description, a.name as account_name
		FROM transactions t LEFT JOIN accounts a ON t.account_id = a.id
		ORDER BY t.created_at DESC LIMIT 5`)
	if err != nil {
		return DashboardData{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var tp, desc, date, acct *string
		var amount int
		if err := rows.Scan(&id, &tp, &amount, &date, &desc, &acct); err != nil {
			return DashboardData{}, err
		}
		d.RecentTransactions = append(d.RecentTransactions, map[string]any{
			"id":               id,
			"type":             tp,
			"amount":           amount,
			"transaction_date": date,
			"description":      desc,
			"account_name":     acct,
		})
	}
	if err := rows.Err(); err != nil {
		return DashboardData{}, err
	}
	if d.RecentTransactions == nil {
		d.RecentTransactions = []map[string]any{}
	}

	return d, nil
}
