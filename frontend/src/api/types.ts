// Tipos que reflejan las respuestas JSON del backend (Go). Se mantienen
// como espejo deliberado de los `model.go` de cada módulo para que un
// cambio de forma en la API sea fácil de detectar aquí.

export type Role = 'admin' | 'gerente' | 'cajero' | 'mesero' | 'cocina' | 'contador'

export interface User {
  id: number
  name: string
  email: string
  role: Role
  position: string
  salary: number
  hire_date: string
  active: boolean
}

export interface LoginResult {
  token: string
  expires_at: string
  user: User
}

export interface Attendance {
  id: number
  user_id: number
  clock_in: string
  clock_out: string | null
  source: string
}

export interface Incident {
  id: number
  user_id: number
  type: 'falta' | 'retardo' | 'permiso'
  date: string
  note: string
  created_by: number
  created_at: string
}

export interface Category {
  id: number
  name: string
  sort_order: number
}

export interface ModifierOption {
  id: number
  group_id: number
  name: string
  price_delta: number
}

export interface ModifierGroup {
  id: number
  name: string
  required: boolean
  multiple: boolean
  options: ModifierOption[]
}

export interface Product {
  id: number
  category_id: number
  name: string
  price: number
  active: boolean
  modifier_groups: ModifierGroup[]
}

export type TableStatus = 'libre' | 'ocupada' | 'cuenta_solicitada'

export interface Zone {
  id: number
  name: string
}

export interface RestaurantTable {
  id: number
  zone_id: number
  zone_name?: string
  number: string
  capacity: number
  status: TableStatus
  active: boolean
  current_waiter_id: number | null
  current_waiter_name?: string | null
  opened_at: string | null
  occupied_minutes?: number | null
  joined_with: number | null
}

export type OrderType = 'mesa' | 'llevar' | 'domicilio'
export type ItemStatus = 'enviado' | 'preparando' | 'listo' | 'entregado' | 'cancelado'

export interface ItemModifier {
  id: number
  name: string
  price_delta: number
}

export interface OrderItem {
  id: number
  order_id: number
  product_id: number
  product_name: string
  qty: number
  unit_price: number
  notes: string
  tanda: number
  status: ItemStatus
  account_id: number | null
  modifiers: ItemModifier[] | null
  line_total: number
  sent_at: string
  ready_at: string | null
  delivered_at: string | null
  cancelled_reason?: string | null
  cancelled_by?: number | null
  table_number?: string
  order_type?: string
}

export interface Order {
  id: number
  table_id: number | null
  table_number?: string | null
  type: OrderType
  waiter_id: number
  waiter_name?: string
  status: 'abierta' | 'cerrada'
  customer_name: string
  customer_phone: string
  customer_address: string
  opened_at: string
  closed_at: string | null
  items: OrderItem[]
  total: number
}

export type PaymentMethod = 'efectivo' | 'tarjeta_debito' | 'tarjeta_credito' | 'transferencia' | 'vale_despensa'

export interface AccountItem {
  id: number
  product_name: string
  qty: number
  unit_price: number
  notes: string
  line_total: number
  status: ItemStatus
}

export interface Payment {
  id: number
  account_id: number
  method: PaymentMethod
  amount: number
  tip_amount: number
  reference: string
  created_by: number
  created_at: string
}

export interface Discount {
  id: number
  account_id: number
  kind: 'descuento' | 'cortesia'
  amount: number
  percent: number | null
  reason: string
  authorized_by: number
  created_at: string
}

export interface Account {
  id: number
  table_id: number | null
  table_number?: string | null
  status: 'abierta' | 'cerrada'
  split_mode: 'completa' | 'producto'
  subtotal: number
  discount_total: number
  tip_total: number
  total: number
  paid: number
  balance: number
  opened_at: string
  closed_at: string | null
  closed_by: number | null
  items: AccountItem[]
  payments: Payment[]
  discounts: Discount[]
}

export interface Ticket {
  account: Account
  note: string
}

export interface Shift {
  id: number
  cashier_id: number
  cashier_name: string
  opening_fund: number
  opened_at: string
  closed_at: string | null
  counted_cash: number | null
  status: 'abierto' | 'cerrado'
}

export interface MethodTotal {
  method: string
  total: number
}

export interface WaiterTotal {
  waiter_id: number
  waiter_name: string
  total: number
}

export interface CategoryTotal {
  category: string
  total: number
}

export interface CashCutBreakdown {
  by_method: MethodTotal[] | null
  by_waiter: WaiterTotal[] | null
  by_category: CategoryTotal[] | null
  tips_total: number
  discounts_total: number
  sales_total: number
}

export interface CashCut {
  id: number
  shift_id: number
  type: 'x' | 'z'
  breakdown: CashCutBreakdown
  expected_cash: number
  counted_cash: number | null
  difference: number | null
  created_by: number
  created_at: string
}

export interface Expense {
  id: number
  category: string
  description: string
  amount: number
  supplier: string
  date: string
  created_by: number
  created_at: string
}

export interface Purchase {
  id: number
  supplier: string
  description: string
  amount: number
  date: string
  created_by: number
  created_at: string
}

export interface InventoryItem {
  id: number
  name: string
  unit: string
  stock: number
  min_stock: number
  updated_at: string
  low_stock: boolean
}

export interface InventoryMovement {
  id: number
  inventory_item_id: number
  type: 'entrada' | 'salida' | 'merma'
  qty: number
  reason: string
  created_by: number
  created_at: string
}

export interface Invoice {
  id: number
  account_id: number
  rfc: string
  razon_social: string
  uso_cfdi: string
  status: 'pendiente' | 'timbrada_simulada' | 'cancelada'
  uuid_fiscal: string | null
  created_at: string
}

export interface DailySales {
  date: string
  total: number
}

export interface ProductSales {
  product_name: string
  qty: number
  total: number
}

export interface SalesReport {
  from: string
  to: string
  total: number
  by_day: DailySales[] | null
  by_product: ProductSales[] | null
  by_category: CategoryTotal[] | null
}

export interface TaxReport {
  from: string
  to: string
  venta_total: number
  iva_trasladado: number
  isr_retenido: number
  note: string
}

export interface ProfitAndLoss {
  from: string
  to: string
  ingresos: number
  gastos: number
  compras: number
  nomina: number
  utilidad: number
  note: string
}

export interface ProductRank {
  product_name: string
  qty: number
}

export interface ShiftSales {
  shift_id: number
  cashier_name: string
  opened_at: string
  total: number
}

export interface WaiterTip {
  waiter_id: number
  waiter_name: string
  tips: number
}

export interface Dashboard {
  from: string
  to: string
  venta_total: number
  venta_por_metodo: MethodTotal[] | null
  ticket_promedio: number
  producto_mas_vendido: ProductRank | null
  producto_menos_vendido: ProductRank | null
  rotacion_minutos_promedio: number
  propinas_total: number
  propinas_por_mesero: WaiterTip[] | null
  venta_por_mesero: WaiterTotal[] | null
  venta_por_turno: ShiftSales[] | null
  faltantes_sobrantes_acumulado: number
}

export interface AuditEntry {
  id: number
  user_id: number | null
  user_name: string | null
  action: string
  entity: string
  entity_id: number | null
  detail: string
  created_at: string
}

export interface ApiError {
  error: string
}
