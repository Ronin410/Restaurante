import { useState } from 'react'
import type { Product } from '../api/types'
import { formatCurrency } from '../utils/money'
import type { NewItemInput } from '../api/endpoints'
import { Field } from './Field'

export function ProductPickerModal({
  product,
  onClose,
  onAdd,
}: {
  product: Product
  onClose: () => void
  onAdd: (item: NewItemInput) => void
}) {
  const [qty, setQty] = useState(1)
  const [notes, setNotes] = useState('')
  const [selected, setSelected] = useState<Record<number, number[]>>({})

  function toggleOption(groupId: number, optionId: number, multiple: boolean) {
    setSelected((prev) => {
      const current = prev[groupId] ?? []
      if (multiple) {
        const next = current.includes(optionId) ? current.filter((id) => id !== optionId) : [...current, optionId]
        return { ...prev, [groupId]: next }
      }
      return { ...prev, [groupId]: current.includes(optionId) ? [] : [optionId] }
    })
  }

  const missingRequired = product.modifier_groups.some((g) => g.required && (selected[g.id] ?? []).length === 0)

  function handleAdd() {
    const modifierOptionIds = Object.values(selected).flat()
    onAdd({ product_id: product.id, qty, notes, modifier_option_ids: modifierOptionIds })
    onClose()
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>
          {product.name} · {formatCurrency(product.price)}
        </h3>

        {product.modifier_groups.map((group) => (
          <div key={group.id} style={{ marginBottom: '0.75rem' }}>
            <div className="section-title" style={{ marginTop: 0 }}>
              {group.name} {group.required && <span className="muted">(obligatorio)</span>}
            </div>
            <div className="chip-row">
              {group.options.map((opt) => (
                <button
                  key={opt.id}
                  type="button"
                  className={`chip ${(selected[group.id] ?? []).includes(opt.id) ? 'selected' : ''}`}
                  onClick={() => toggleOption(group.id, opt.id, group.multiple)}
                >
                  {opt.name}
                  {opt.price_delta !== 0 && ` (+${formatCurrency(opt.price_delta)})`}
                </button>
              ))}
            </div>
          </div>
        ))}

        <Field label="Cantidad">
          <input type="number" min={1} value={qty} onChange={(e) => setQty(Number(e.target.value))} />
        </Field>
        <Field label='Notas (ej. "sin cebolla", "extra salsa")'>
          <textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
        </Field>

        <div className="toolbar">
          <button className="btn primary" disabled={missingRequired || qty <= 0} onClick={handleAdd}>
            Agregar al pedido
          </button>
          <button className="btn" onClick={onClose}>
            Cancelar
          </button>
        </div>
        {missingRequired && <p className="muted">Selecciona una opción de cada modificador obligatorio.</p>}
      </div>
    </div>
  )
}
