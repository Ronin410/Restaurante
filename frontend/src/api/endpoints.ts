// Funciones tipadas por módulo que envuelven al cliente HTTP. Mantenerlas
// aquí (en vez de llamar apiFetch directo desde los componentes) deja un
// solo lugar por endpoint si cambia la forma de la API.

import { apiFetch, apiMutateOrQueue, type QueueableResult } from './client'
import type {
  Account,
  AuditEntry,
  CashCut,
  Category,
  Dashboard,
  Discount,
  Expense,
  Incident,
  InventoryItem,
  InventoryMovement,
  Invoice,
  LoginResult,
  ModifierGroup,
  Order,
  OrderItem,
  Payment,
  ProfitAndLoss,
  Product,
  Purchase,
  RestaurantTable,
  SalesReport,
  Shift,
  TaxReport,
  Ticket,
  User,
  Zone,
} from './types'

// ---- Auth / Personal ----

export const AuthApi = {
  login: (email: string, password: string) =>
    apiFetch<LoginResult>('/auth/login', { method: 'POST', body: { email, password } }),
  me: () => apiFetch<User>('/auth/me'),
}

export const EmployeesApi = {
  list: (activeOnly = false) => apiFetch<User[]>(`/employees/?active=${activeOnly ? '1' : '0'}`),
  create: (input: {
    name: string
    email: string
    password: string
    role: string
    position: string
    salary: number
    hire_date?: string
  }) => apiFetch<User>('/employees/', { method: 'POST', body: input }),
  update: (id: number, input: { name: string; role: string; position: string; salary: number; active: boolean }) =>
    apiFetch<User>(`/employees/${id}`, { method: 'PUT', body: input }),
  clockIn: (id: number) => apiFetch(`/employees/${id}/attendance/clock-in`, { method: 'POST' }),
  clockOut: (id: number) => apiFetch(`/employees/${id}/attendance/clock-out`, { method: 'POST' }),
  attendance: (id: number) => apiFetch(`/employees/${id}/attendance`),
  incidents: (id: number) => apiFetch<Incident[]>(`/employees/${id}/incidents`),
  createIncident: (id: number, input: { type: string; date: string; note: string }) =>
    apiFetch<Incident>(`/employees/${id}/incidents`, { method: 'POST', body: input }),
}

// ---- Catálogo ----

export const CatalogApi = {
  categories: () => apiFetch<Category[]>('/categories/'),
  createCategory: (input: { name: string; sort_order: number }) =>
    apiFetch<Category>('/categories/', { method: 'POST', body: input }),
  products: (activeOnly = false) => apiFetch<Product[]>(`/products/?active=${activeOnly ? '1' : '0'}`),
  createProduct: (input: { category_id: number; name: string; price: number; active: boolean; group_ids: number[] }) =>
    apiFetch<Product>('/products/', { method: 'POST', body: input }),
  updateProduct: (
    id: number,
    input: { category_id: number; name: string; price: number; active: boolean; group_ids: number[] },
  ) => apiFetch<Product>(`/products/${id}`, { method: 'PUT', body: input }),
  modifierGroups: () => apiFetch<ModifierGroup[]>('/modifier-groups/'),
  createModifierGroup: (input: { name: string; required: boolean; multiple: boolean }) =>
    apiFetch<ModifierGroup>('/modifier-groups/', { method: 'POST', body: input }),
  addModifierOption: (groupId: number, input: { name: string; price_delta: number }) =>
    apiFetch(`/modifier-groups/${groupId}/options`, { method: 'POST', body: input }),
}

// ---- Mesas ----

export const TablesApi = {
  zones: () => apiFetch<Zone[]>('/zones/'),
  createZone: (name: string) => apiFetch<Zone>('/zones/', { method: 'POST', body: { name } }),
  updateZone: (id: number, name: string) => apiFetch<Zone>(`/zones/${id}`, { method: 'PUT', body: { name } }),
  deleteZone: (id: number) => apiFetch<void>(`/zones/${id}`, { method: 'DELETE' }),
  list: (includeInactive = false) => apiFetch<RestaurantTable[]>(`/tables/${includeInactive ? '?include_inactive=1' : ''}`),
  get: (id: number) => apiFetch<RestaurantTable>(`/tables/${id}`),
  create: (input: { zone_id: number; number: string; capacity: number }) =>
    apiFetch<RestaurantTable>('/tables/', { method: 'POST', body: input }),
  update: (id: number, input: { zone_id: number; number: string; capacity: number }) =>
    apiFetch<RestaurantTable>(`/tables/${id}`, { method: 'PUT', body: input }),
  remove: (id: number) => apiFetch<RestaurantTable>(`/tables/${id}`, { method: 'DELETE' }),
  reactivate: (id: number) => apiFetch<RestaurantTable>(`/tables/${id}/reactivate`, { method: 'POST' }),
  open: (id: number, waiterId?: number) =>
    apiFetch<RestaurantTable>(`/tables/${id}/open`, { method: 'POST', body: waiterId ? { waiter_id: waiterId } : {} }),
  setStatus: (id: number, status: string) =>
    apiFetch<RestaurantTable>(`/tables/${id}/status`, { method: 'POST', body: { status } }),
  assign: (id: number, waiterId: number) =>
    apiFetch<RestaurantTable>(`/tables/${id}/assign`, { method: 'POST', body: { waiter_id: waiterId } }),
  join: (id: number, withId: number) =>
    apiFetch<RestaurantTable>(`/tables/${id}/join/${withId}`, { method: 'POST' }),
  unjoin: (id: number) => apiFetch<RestaurantTable>(`/tables/${id}/unjoin`, { method: 'POST' }),
  transfer: (id: number, toWaiterId: number) =>
    apiFetch(`/tables/${id}/transfer`, { method: 'POST', body: { to_waiter_id: toWaiterId } }),
}

// ---- Comandas ----

export interface NewItemInput {
  product_id: number
  qty: number
  notes?: string
  modifier_option_ids?: number[]
}

export const OrdersApi = {
  listOpen: (tableId?: number) => apiFetch<Order[]>(`/orders/${tableId ? `?table_id=${tableId}` : ''}`),
  get: (id: number) => apiFetch<Order>(`/orders/${id}`),
  create: (input: {
    table_id?: number | null
    type: string
    customer_name?: string
    customer_phone?: string
    customer_address?: string
    items: NewItemInput[]
  }): Promise<QueueableResult<Order>> => apiMutateOrQueue('/orders/', input, `Comanda mesa ${input.table_id ?? ''}`),
  addItems: (orderId: number, items: NewItemInput[]): Promise<QueueableResult<Order>> =>
    apiMutateOrQueue(`/orders/${orderId}/items`, { items }, `Agregar productos a comanda #${orderId}`),
  close: (id: number) => apiFetch(`/orders/${id}/close`, { method: 'POST' }),
  kdsItems: () => apiFetch<OrderItem[]>('/kds/items'),
  updateItemStatus: (itemId: number, status: string) =>
    apiFetch<OrderItem>(`/orders/items/${itemId}/status`, { method: 'POST', body: { status } }),
  cancelItem: (itemId: number, reason: string) =>
    apiFetch<OrderItem>(`/orders/items/${itemId}/cancel`, { method: 'POST', body: { reason } }),
}

// ---- Cuentas y cobro ----

export const AccountsApi = {
  generate: (tableId: number) => apiFetch<Account>('/accounts/generate', { method: 'POST', body: { table_id: tableId } }),
  splitByProduct: (tableId: number, groups: number[][]) =>
    apiFetch<Account[]>('/accounts/split-by-product', { method: 'POST', body: { table_id: tableId, groups } }),
  get: (id: number) => apiFetch<Account>(`/accounts/${id}`),
  ticket: (id: number) => apiFetch<Ticket>(`/accounts/${id}/ticket`),
  byTable: (tableId: number) => apiFetch<Account[]>(`/tables/${tableId}/accounts`),
  addDiscount: (id: number, input: { kind: string; amount?: number; percent?: number; reason: string }) =>
    apiFetch<Account>(`/accounts/${id}/discounts`, { method: 'POST', body: input }),
  addPayment: (
    id: number,
    input: { method: string; amount: number; tip_amount?: number; reference?: string },
  ): Promise<QueueableResult<Account>> => apiMutateOrQueue(`/accounts/${id}/payments`, input, `Pago cuenta #${id}`),
  close: (id: number) => apiFetch<Account>(`/accounts/${id}/close`, { method: 'POST' }),
}

// ---- Cortes de caja ----

export const CashCutsApi = {
  openShift: (openingFund: number, cashierId?: number) =>
    apiFetch<Shift>('/shifts/open', { method: 'POST', body: { opening_fund: openingFund, cashier_id: cashierId ?? 0 } }),
  listShifts: (status?: string) => apiFetch<Shift[]>(`/shifts/${status ? `?status=${status}` : ''}`),
  getShift: (id: number) => apiFetch<Shift>(`/shifts/${id}`),
  closeShift: (id: number, countedCash: number) =>
    apiFetch<{ shift: Shift; cut: CashCut }>(`/shifts/${id}/close`, { method: 'POST', body: { counted_cash: countedCash } }),
  xCut: (shiftId: number, countedCash?: number) =>
    apiFetch<CashCut>(`/shifts/${shiftId}/cuts/x`, {
      method: 'POST',
      body: countedCash !== undefined ? { counted_cash: countedCash } : {},
    }),
  shiftCuts: (shiftId: number) => apiFetch<CashCut[]>(`/shifts/${shiftId}/cuts`),
  listCuts: (from?: string, to?: string, type?: string) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from)
    if (to) params.set('to', to)
    if (type) params.set('type', type)
    const qs = params.toString()
    return apiFetch<CashCut[]>(`/cuts/${qs ? `?${qs}` : ''}`)
  },
  exportUrl: (cutId: number) => `/cuts/${cutId}/export`,
}

// ---- Contaduría ----

export const AccountingApi = {
  expenses: (from?: string, to?: string) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from)
    if (to) params.set('to', to)
    return apiFetch<Expense[]>(`/expenses/?${params.toString()}`)
  },
  createExpense: (input: { category: string; description: string; amount: number; supplier?: string; date?: string }) =>
    apiFetch<Expense>('/expenses/', { method: 'POST', body: input }),
  updateExpense: (id: number, input: { category: string; description: string; amount: number; supplier?: string; date?: string }) =>
    apiFetch<Expense>(`/expenses/${id}`, { method: 'PUT', body: input }),
  deleteExpense: (id: number) => apiFetch<void>(`/expenses/${id}`, { method: 'DELETE' }),
  purchases: (from?: string, to?: string) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from)
    if (to) params.set('to', to)
    return apiFetch<Purchase[]>(`/purchases/?${params.toString()}`)
  },
  createPurchase: (input: { supplier: string; description: string; amount: number; date?: string }) =>
    apiFetch<Purchase>('/purchases/', { method: 'POST', body: input }),
  updatePurchase: (id: number, input: { supplier: string; description: string; amount: number; date?: string }) =>
    apiFetch<Purchase>(`/purchases/${id}`, { method: 'PUT', body: input }),
  deletePurchase: (id: number) => apiFetch<void>(`/purchases/${id}`, { method: 'DELETE' }),
  inventory: () => apiFetch<InventoryItem[]>('/inventory/'),
  updateInventoryItem: (id: number, input: { name: string; unit: string; min_stock: number }) =>
    apiFetch<InventoryItem>(`/inventory/${id}`, { method: 'PUT', body: input }),
  deleteInventoryItem: (id: number) => apiFetch<void>(`/inventory/${id}`, { method: 'DELETE' }),
  deleteMovement: (itemId: number, movementId: number) =>
    apiFetch<InventoryItem>(`/inventory/${itemId}/movements/${movementId}`, { method: 'DELETE' }),
  cancelInvoice: (id: number) => apiFetch<Invoice>(`/invoices/${id}/cancel`, { method: 'POST' }),
  createInventoryItem: (input: { name: string; unit: string; min_stock: number }) =>
    apiFetch<InventoryItem>('/inventory/', { method: 'POST', body: input }),
  addMovement: (itemId: number, input: { type: string; qty: number; reason: string }) =>
    apiFetch<InventoryItem>(`/inventory/${itemId}/movements`, { method: 'POST', body: input }),
  movements: (itemId: number) => apiFetch<InventoryMovement[]>(`/inventory/${itemId}/movements`),
  invoices: () => apiFetch<Invoice[]>('/invoices/'),
  createInvoice: (input: { account_id: number; rfc: string; razon_social: string; uso_cfdi?: string }) =>
    apiFetch<Invoice>('/invoices/', { method: 'POST', body: input }),
  salesReport: (from?: string, to?: string) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from)
    if (to) params.set('to', to)
    return apiFetch<SalesReport>(`/reports/sales?${params.toString()}`)
  },
  taxReport: (from?: string, to?: string) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from)
    if (to) params.set('to', to)
    return apiFetch<TaxReport>(`/reports/taxes?${params.toString()}`)
  },
  pnl: (from?: string, to?: string) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from)
    if (to) params.set('to', to)
    return apiFetch<ProfitAndLoss>(`/reports/pnl?${params.toString()}`)
  },
}

export const KpisApi = {
  dashboard: (from?: string, to?: string) => {
    const params = new URLSearchParams()
    if (from) params.set('from', from)
    if (to) params.set('to', to)
    return apiFetch<Dashboard>(`/kpis/dashboard?${params.toString()}`)
  },
}

export const AuditApi = {
  list: (entity?: string) => apiFetch<AuditEntry[]>(`/audit-log${entity ? `?entity=${entity}` : ''}`),
}

export type { Payment, Discount }
