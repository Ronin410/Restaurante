package accounting

type Expense struct {
	ID          int     `json:"id"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Supplier    string  `json:"supplier"`
	Date        string  `json:"date"`
	CreatedBy   int     `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
}

type ExpenseInput struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Supplier    string  `json:"supplier"`
	Date        string  `json:"date"`
}

type Purchase struct {
	ID          int     `json:"id"`
	Supplier    string  `json:"supplier"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
	CreatedBy   int     `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
}

type PurchaseInput struct {
	Supplier    string  `json:"supplier"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
}

type InventoryItem struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Stock     float64 `json:"stock"`
	MinStock  float64 `json:"min_stock"`
	UpdatedAt string  `json:"updated_at"`
	LowStock  bool    `json:"low_stock"`
}

type InventoryItemInput struct {
	Name     string  `json:"name"`
	Unit     string  `json:"unit"`
	MinStock float64 `json:"min_stock"`
}

type InventoryMovementInput struct {
	Type   string  `json:"type"` // entrada | salida | merma
	Qty    float64 `json:"qty"`
	Reason string  `json:"reason"`
}

type InventoryMovement struct {
	ID              int     `json:"id"`
	InventoryItemID int     `json:"inventory_item_id"`
	Type            string  `json:"type"`
	Qty             float64 `json:"qty"`
	Reason          string  `json:"reason"`
	CreatedBy       int     `json:"created_by"`
	CreatedAt       string  `json:"created_at"`
}

// Invoice es un stub de facturación CFDI 4.0. La integración real requiere
// contratar un PAC (Proveedor Autorizado de Certificación) del SAT y sus
// credenciales; aquí se deja el flujo y modelo de datos listos, con
// "timbrado simulado" para que el resto del sistema (UI, reportes) pueda
// desarrollarse y probarse sin depender de ese proveedor externo.
type Invoice struct {
	ID          int     `json:"id"`
	AccountID   int     `json:"account_id"`
	RFC         string  `json:"rfc"`
	RazonSocial string  `json:"razon_social"`
	UsoCFDI     string  `json:"uso_cfdi"`
	Status      string  `json:"status"` // pendiente | timbrada_simulada | cancelada
	UUIDFiscal  *string `json:"uuid_fiscal"`
	CreatedAt   string  `json:"created_at"`
}

type InvoiceInput struct {
	AccountID   int    `json:"account_id"`
	RFC         string `json:"rfc"`
	RazonSocial string `json:"razon_social"`
	UsoCFDI     string `json:"uso_cfdi"`
}

type DailySales struct {
	Date  string  `json:"date"`
	Total float64 `json:"total"`
}

type ProductSales struct {
	ProductName string  `json:"product_name"`
	Qty         float64 `json:"qty"`
	Total       float64 `json:"total"`
}

type CategorySales struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

type SalesReport struct {
	From       string          `json:"from"`
	To         string          `json:"to"`
	Total      float64         `json:"total"`
	ByDay      []DailySales    `json:"by_day"`
	ByProduct  []ProductSales  `json:"by_product"`
	ByCategory []CategorySales `json:"by_category"`
}

type TaxReport struct {
	From          string  `json:"from"`
	To            string  `json:"to"`
	VentaTotal    float64 `json:"venta_total"`
	IVATrasladado float64 `json:"iva_trasladado"`
	ISRRetenido   float64 `json:"isr_retenido"`
	Note          string  `json:"note"`
}

type ProfitAndLoss struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Ingresos float64 `json:"ingresos"`
	Gastos   float64 `json:"gastos"`
	Compras  float64 `json:"compras"`
	Nomina   float64 `json:"nomina"`
	Utilidad float64 `json:"utilidad"`
	Note     string  `json:"note"`
}
