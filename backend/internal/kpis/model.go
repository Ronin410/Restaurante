package kpis

type MethodTotal struct {
	Method string  `json:"method"`
	Total  float64 `json:"total"`
}

type WaiterTotal struct {
	WaiterID   int     `json:"waiter_id"`
	WaiterName string  `json:"waiter_name"`
	Total      float64 `json:"total"`
}

type WaiterTip struct {
	WaiterID   int     `json:"waiter_id"`
	WaiterName string  `json:"waiter_name"`
	Tips       float64 `json:"tips"`
}

type ProductRank struct {
	ProductName string  `json:"product_name"`
	Qty         float64 `json:"qty"`
}

type ShiftSales struct {
	ShiftID     int     `json:"shift_id"`
	CashierName string  `json:"cashier_name"`
	OpenedAt    string  `json:"opened_at"`
	Total       float64 `json:"total"`
}

type Dashboard struct {
	From                 string        `json:"from"`
	To                   string        `json:"to"`
	VentaTotal           float64       `json:"venta_total"`
	VentaPorMetodo       []MethodTotal `json:"venta_por_metodo"`
	TicketPromedio       float64       `json:"ticket_promedio"`
	ProductoMasVendido   *ProductRank  `json:"producto_mas_vendido"`
	ProductoMenosVendido *ProductRank  `json:"producto_menos_vendido"`
	RotacionMinutosProm  float64       `json:"rotacion_minutos_promedio"`
	PropinasTotal        float64       `json:"propinas_total"`
	PropinasPorMesero    []WaiterTip   `json:"propinas_por_mesero"`
	VentaPorMesero       []WaiterTotal `json:"venta_por_mesero"`
	VentaPorTurno        []ShiftSales  `json:"venta_por_turno"`
	FaltantesSobrantes   float64       `json:"faltantes_sobrantes_acumulado"`
}
