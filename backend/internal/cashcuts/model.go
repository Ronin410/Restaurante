package cashcuts

type Shift struct {
	ID          int      `json:"id"`
	CashierID   int      `json:"cashier_id"`
	CashierName string   `json:"cashier_name,omitempty"`
	OpeningFund float64  `json:"opening_fund"`
	OpenedAt    string   `json:"opened_at"`
	ClosedAt    *string  `json:"closed_at"`
	CountedCash *float64 `json:"counted_cash"`
	Status      string   `json:"status"` // abierto | cerrado
}

type OpenShiftInput struct {
	OpeningFund float64 `json:"opening_fund"`
	CashierID   int     `json:"cashier_id"` // opcional: si un gerente abre turno a nombre de un cajero
}

type CloseShiftInput struct {
	CountedCash float64 `json:"counted_cash"`
}

type MethodTotal struct {
	Method string  `json:"method"`
	Total  float64 `json:"total"`
}

type WaiterTotal struct {
	WaiterID   int     `json:"waiter_id"`
	WaiterName string  `json:"waiter_name"`
	Total      float64 `json:"total"`
}

type CategoryTotal struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

type Breakdown struct {
	ByMethod       []MethodTotal   `json:"by_method"`
	ByWaiter       []WaiterTotal   `json:"by_waiter"`
	ByCategory     []CategoryTotal `json:"by_category"`
	TipsTotal      float64         `json:"tips_total"`
	DiscountsTotal float64         `json:"discounts_total"`
	SalesTotal     float64         `json:"sales_total"`
}

type CashCut struct {
	ID           int       `json:"id"`
	ShiftID      int       `json:"shift_id"`
	Type         string    `json:"type"` // x | z
	Breakdown    Breakdown `json:"breakdown"`
	ExpectedCash float64   `json:"expected_cash"`
	CountedCash  *float64  `json:"counted_cash"`
	Difference   *float64  `json:"difference"`
	CreatedBy    int       `json:"created_by"`
	CreatedAt    string    `json:"created_at"`
}

type CutInput struct {
	CountedCash *float64 `json:"counted_cash"`
}

type CloseShiftResult struct {
	Shift Shift   `json:"shift"`
	Cut   CashCut `json:"cut"`
}
