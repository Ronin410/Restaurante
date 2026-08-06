import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ProductPickerModal } from './ProductPickerModal'
import type { Product } from '../api/types'

const tacoProduct: Product = {
  id: 1,
  category_id: 1,
  name: 'Taco',
  price: 18,
  active: true,
  modifier_groups: [
    {
      id: 1,
      name: 'Guisado',
      required: true,
      multiple: false,
      options: [
        { id: 1, group_id: 1, name: 'Pastor', price_delta: 0 },
        { id: 2, group_id: 1, name: 'Suadero', price_delta: 0 },
      ],
    },
    {
      id: 2,
      name: 'Extras',
      required: false,
      multiple: true,
      options: [{ id: 3, group_id: 2, name: 'Doble queso', price_delta: 8 }],
    },
  ],
}

const refrescoProduct: Product = {
  id: 2,
  category_id: 2,
  name: 'Refresco',
  price: 25,
  active: true,
  modifier_groups: [],
}

describe('ProductPickerModal', () => {
  it('bloquea agregar hasta elegir el modificador obligatorio', async () => {
    const onAdd = vi.fn()
    render(<ProductPickerModal product={tacoProduct} onClose={vi.fn()} onAdd={onAdd} />)

    const addButton = screen.getByRole('button', { name: 'Agregar al pedido' })
    expect(addButton).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: 'Pastor' }))
    expect(addButton).toBeEnabled()
  })

  it('agrega el producto con los modificadores seleccionados y la cantidad indicada', async () => {
    const onAdd = vi.fn()
    const onClose = vi.fn()
    render(<ProductPickerModal product={tacoProduct} onClose={onClose} onAdd={onAdd} />)

    await userEvent.click(screen.getByRole('button', { name: 'Pastor' }))
    await userEvent.click(screen.getByRole('button', { name: /Doble queso/ }))

    const qtyInput = screen.getByLabelText('Cantidad')
    await userEvent.clear(qtyInput)
    await userEvent.type(qtyInput, '3')

    await userEvent.click(screen.getByRole('button', { name: 'Agregar al pedido' }))

    expect(onAdd).toHaveBeenCalledWith({
      product_id: 1,
      qty: 3,
      notes: '',
      modifier_option_ids: [1, 3],
    })
    expect(onClose).toHaveBeenCalled()
  })

  it('un producto sin modificadores se puede agregar directamente', async () => {
    const onAdd = vi.fn()
    render(<ProductPickerModal product={refrescoProduct} onClose={vi.fn()} onAdd={onAdd} />)

    const addButton = screen.getByRole('button', { name: 'Agregar al pedido' })
    expect(addButton).toBeEnabled()
    await userEvent.click(addButton)
    expect(onAdd).toHaveBeenCalledWith({ product_id: 2, qty: 1, notes: '', modifier_option_ids: [] })
  })

  it('un grupo de selección única deselecciona la opción anterior', async () => {
    const onAdd = vi.fn()
    render(<ProductPickerModal product={tacoProduct} onClose={vi.fn()} onAdd={onAdd} />)

    await userEvent.click(screen.getByRole('button', { name: 'Pastor' }))
    await userEvent.click(screen.getByRole('button', { name: 'Suadero' }))
    await userEvent.click(screen.getByRole('button', { name: 'Agregar al pedido' }))

    expect(onAdd).toHaveBeenCalledWith(expect.objectContaining({ modifier_option_ids: [2] }))
  })
})
