package accounts

// AccountItem es una vista de solo lectura de las partidas (order_items)
// asignadas a la cuenta; el detalle completo de la comanda vive en el
// módulo orders.
type AccountItem struct {
	ID          int     `json:"id"`
	ProductName string  `json:"product_name"`
	Qty         float64 `json:"qty"`
	UnitPrice   float64 `json:"unit_price"`
	Notes       string  `json:"notes"`
	LineTotal   float64 `json:"line_total"`
	Status      string  `json:"status"`
}

type Payment struct {
	ID        int     `json:"id"`
	AccountID int     `json:"account_id"`
	Method    string  `json:"method"`
	Amount    float64 `json:"amount"`
	TipAmount float64 `json:"tip_amount"`
	Reference string  `json:"reference"`
	CreatedBy int     `json:"created_by"`
	CreatedAt string  `json:"created_at"`
}

type Discount struct {
	ID           int      `json:"id"`
	AccountID    int      `json:"account_id"`
	Kind         string   `json:"kind"` // descuento | cortesia
	Amount       float64  `json:"amount"`
	Percent      *float64 `json:"percent"`
	Reason       string   `json:"reason"`
	AuthorizedBy int      `json:"authorized_by"`
	CreatedAt    string   `json:"created_at"`
}

type Account struct {
	ID            int           `json:"id"`
	TableID       *int          `json:"table_id"`
	TableNumber   *string       `json:"table_number,omitempty"`
	Status        string        `json:"status"`
	SplitMode     string        `json:"split_mode"`
	Subtotal      float64       `json:"subtotal"`
	DiscountTotal float64       `json:"discount_total"`
	TipTotal      float64       `json:"tip_total"`
	Total         float64       `json:"total"`
	Paid          float64       `json:"paid"`
	Balance       float64       `json:"balance"`
	OpenedAt      string        `json:"opened_at"`
	ClosedAt      *string       `json:"closed_at"`
	ClosedBy      *int          `json:"closed_by"`
	Items         []AccountItem `json:"items"`
	Payments      []Payment     `json:"payments"`
	Discounts     []Discount    `json:"discounts"`
}

type GenerateAccountInput struct {
	TableID int `json:"table_id"`
}

type SplitByProductInput struct {
	TableID int     `json:"table_id"`
	Groups  [][]int `json:"groups"`
}

type PaymentInput struct {
	Method    string  `json:"method"`
	Amount    float64 `json:"amount"`
	TipAmount float64 `json:"tip_amount"`
	Reference string  `json:"reference"`
}

type DiscountInput struct {
	Kind    string   `json:"kind"` // descuento | cortesia
	Amount  float64  `json:"amount"`
	Percent *float64 `json:"percent"`
	Reason  string   `json:"reason"`
}

// Ticket es la representación imprimible/reimprimible de una cuenta.
type Ticket struct {
	Account Account `json:"account"`
	Note    string  `json:"note"`
}
