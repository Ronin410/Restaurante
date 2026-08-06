package orders

type ItemModifier struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	PriceDelta float64 `json:"price_delta"`
}

type OrderItem struct {
	ID              int            `json:"id"`
	OrderID         int            `json:"order_id"`
	ProductID       int            `json:"product_id"`
	ProductName     string         `json:"product_name"`
	Qty             float64        `json:"qty"`
	UnitPrice       float64        `json:"unit_price"`
	Notes           string         `json:"notes"`
	Tanda           int            `json:"tanda"`
	Status          string         `json:"status"` // enviado | preparando | listo | entregado | cancelado
	AccountID       *int           `json:"account_id"`
	Modifiers       []ItemModifier `json:"modifiers"`
	LineTotal       float64        `json:"line_total"`
	SentAt          string         `json:"sent_at"`
	ReadyAt         *string        `json:"ready_at"`
	DeliveredAt     *string        `json:"delivered_at"`
	CancelledReason *string        `json:"cancelled_reason,omitempty"`
	CancelledBy     *int           `json:"cancelled_by,omitempty"`
	// Contexto de mesa/mesero para las vistas de cocina (KDS).
	TableNumber string `json:"table_number,omitempty"`
	OrderType   string `json:"order_type,omitempty"`
}

type Order struct {
	ID              int         `json:"id"`
	TableID         *int        `json:"table_id"`
	TableNumber     *string     `json:"table_number,omitempty"`
	Type            string      `json:"type"` // mesa | llevar | domicilio
	WaiterID        int         `json:"waiter_id"`
	WaiterName      string      `json:"waiter_name,omitempty"`
	Status          string      `json:"status"` // abierta | cerrada
	CustomerName    string      `json:"customer_name"`
	CustomerPhone   string      `json:"customer_phone"`
	CustomerAddress string      `json:"customer_address"`
	OpenedAt        string      `json:"opened_at"`
	ClosedAt        *string     `json:"closed_at"`
	Items           []OrderItem `json:"items"`
	Total           float64     `json:"total"`
}

type NewItemInput struct {
	ProductID         int     `json:"product_id"`
	Qty               float64 `json:"qty"`
	Notes             string  `json:"notes"`
	ModifierOptionIDs []int   `json:"modifier_option_ids"`
}

type CreateOrderInput struct {
	TableID         *int           `json:"table_id"`
	Type            string         `json:"type"`
	CustomerName    string         `json:"customer_name"`
	CustomerPhone   string         `json:"customer_phone"`
	CustomerAddress string         `json:"customer_address"`
	Items           []NewItemInput `json:"items"`
}

type AddItemsInput struct {
	Items []NewItemInput `json:"items"`
}

type UpdateItemStatusInput struct {
	Status string `json:"status"`
}

type CancelItemInput struct {
	Reason string `json:"reason"`
}
