package models

import "time"

// Payout represents a platform payout record.
type Payout struct {
	ID                    int       `json:"id"`
	OutletName            string    `json:"outlet_name"`
	Platform              string    `json:"platform"` // Swiggy, Zomato
	PeriodStart           *string   `json:"period_start"`
	PeriodEnd             *string   `json:"period_end"`
	SettlementDate        *string   `json:"settlement_date"`
	TotalOrders           int       `json:"total_orders"`
	GrossSalesAmt         int       `json:"gross_sales_amt"`
	RestaurantDiscountAmt int       `json:"restaurant_discount_amt"`
	PlatformCommissionAmt int       `json:"platform_commission_amt"`
	TaxesTcsTdsAmt        int       `json:"taxes_tcs_tds_amt"`
	MarketingAdsAmt       int       `json:"marketing_ads_amt"`
	FinalPayoutAmt        int       `json:"final_payout_amt"`
	UtrNumber             string    `json:"utr_number"`
	CreatedAt             time.Time `json:"created_at"`
	// Computed fields
	Allocated   int `json:"allocated"`
	Unallocated int `json:"unallocated"`
}

// PayoutInput is used for creating/updating payout records.
type PayoutInput struct {
	OutletName            string  `json:"outlet_name"`
	Platform              string  `json:"platform"`
	PeriodStart           *string `json:"period_start"`
	PeriodEnd             *string `json:"period_end"`
	SettlementDate        *string `json:"settlement_date"`
	TotalOrders           int     `json:"total_orders"`
	GrossSalesAmt         int     `json:"gross_sales_amt"`
	RestaurantDiscountAmt int     `json:"restaurant_discount_amt"`
	PlatformCommissionAmt int     `json:"platform_commission_amt"`
	TaxesTcsTdsAmt        int     `json:"taxes_tcs_tds_amt"`
	MarketingAdsAmt       int     `json:"marketing_ads_amt"`
	FinalPayoutAmt        int     `json:"final_payout_amt"`
	UtrNumber             string  `json:"utr_number"`
}

func (p *PayoutInput) Validate() string {
	if p.OutletName == "" {
		return "outlet_name is required"
	}
	switch p.Platform {
	case "Swiggy", "Zomato":
	default:
		return "platform must be Swiggy or Zomato"
	}
	return ""
}
