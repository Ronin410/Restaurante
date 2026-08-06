import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CatalogApi } from '../api/endpoints'
import { formatCurrency } from '../utils/money'
import { Field } from '../components/Field'

export function CatalogPage() {
  const queryClient = useQueryClient()
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: CatalogApi.categories })
  const { data: products } = useQuery({ queryKey: ['products-all'], queryFn: () => CatalogApi.products(false) })
  const { data: groups } = useQuery({ queryKey: ['modifier-groups'], queryFn: CatalogApi.modifierGroups })

  const [newCategory, setNewCategory] = useState('')
  const [newProduct, setNewProduct] = useState({ name: '', price: '', category_id: '' })

  const createCategory = useMutation({
    mutationFn: () => CatalogApi.createCategory({ name: newCategory, sort_order: (categories?.length ?? 0) + 1 }),
    onSuccess: () => {
      setNewCategory('')
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })

  const createProduct = useMutation({
    mutationFn: () =>
      CatalogApi.createProduct({
        name: newProduct.name,
        price: Number(newProduct.price),
        category_id: Number(newProduct.category_id),
        active: true,
        group_ids: [],
      }),
    onSuccess: () => {
      setNewProduct({ name: '', price: '', category_id: '' })
      queryClient.invalidateQueries({ queryKey: ['products-all'] })
    },
  })

  return (
    <div>
      <h2>Catálogo</h2>
      <div className="grid" style={{ gridTemplateColumns: '1fr 1fr', alignItems: 'start' }}>
        <div className="card">
          <h3 style={{ marginTop: 0 }}>Categorías</h3>
          <ul>
            {(categories ?? []).map((c) => (
              <li key={c.id}>{c.name}</li>
            ))}
          </ul>
          <div className="toolbar">
            <input placeholder="Nueva categoría" value={newCategory} onChange={(e) => setNewCategory(e.target.value)} />
            <button className="btn" disabled={!newCategory} onClick={() => createCategory.mutate()}>
              Agregar
            </button>
          </div>

          <h3>Grupos de modificadores</h3>
          {(groups ?? []).map((g) => (
            <div key={g.id} style={{ marginBottom: '0.5rem' }}>
              <strong>{g.name}</strong> {g.required && <span className="muted">(obligatorio)</span>}
              <div className="chip-row">
                {g.options.map((o) => (
                  <span key={o.id} className="chip">
                    {o.name}
                    {o.price_delta !== 0 && ` +${formatCurrency(o.price_delta)}`}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="card">
          <h3 style={{ marginTop: 0 }}>Productos</h3>
          <table className="data-table">
            <thead>
              <tr>
                <th>Nombre</th>
                <th>Categoría</th>
                <th>Precio</th>
                <th>Activo</th>
              </tr>
            </thead>
            <tbody>
              {(products ?? []).map((p) => (
                <tr key={p.id}>
                  <td>{p.name}</td>
                  <td>{categories?.find((c) => c.id === p.category_id)?.name}</td>
                  <td>{formatCurrency(p.price)}</td>
                  <td>{p.active ? 'Sí' : 'No'}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <h4>Nuevo producto</h4>
          <Field label="Nombre">
            <input value={newProduct.name} onChange={(e) => setNewProduct({ ...newProduct, name: e.target.value })} />
          </Field>
          <Field label="Categoría">
            <select
              value={newProduct.category_id}
              onChange={(e) => setNewProduct({ ...newProduct, category_id: e.target.value })}
            >
              <option value="">Selecciona…</option>
              {(categories ?? []).map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Precio">
            <input type="number" value={newProduct.price} onChange={(e) => setNewProduct({ ...newProduct, price: e.target.value })} />
          </Field>
          {createProduct.isError && <div className="error-banner">{(createProduct.error as Error).message}</div>}
          <button
            className="btn primary"
            disabled={!newProduct.name || !newProduct.price || !newProduct.category_id}
            onClick={() => createProduct.mutate()}
          >
            Crear producto
          </button>
        </div>
      </div>
    </div>
  )
}
