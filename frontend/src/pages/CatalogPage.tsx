import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CatalogApi } from '../api/endpoints'
import type { Category, ModifierGroup, Product } from '../api/types'
import { formatCurrency } from '../utils/money'
import { Field } from '../components/Field'

type Tab = 'productos' | 'modificadores'

export function CatalogPage() {
  const [tab, setTab] = useState<Tab>('productos')
  return (
    <div>
      <div className="page-header">
        <div>
          <h2>Catálogo</h2>
          <p className="muted">Lo que se vende: categorías, productos y las opciones que el cliente puede elegir.</p>
        </div>
      </div>
      <div className="segmented" style={{ marginBottom: '1rem' }}>
        <button className={tab === 'productos' ? 'active' : ''} onClick={() => setTab('productos')}>
          Productos
        </button>
        <button className={tab === 'modificadores' ? 'active' : ''} onClick={() => setTab('modificadores')}>
          Modificadores (guisados, extras…)
        </button>
      </div>
      {tab === 'productos' ? <ProductsTab /> : <ModifiersTab />}
    </div>
  )
}

/** Ventana pequeña con un solo formulario. */
function Dialog({
  title,
  onClose,
  onSubmit,
  submitLabel = 'Guardar',
  disabled,
  error,
  pending,
  children,
}: {
  title: string
  onClose: () => void
  onSubmit: () => void
  submitLabel?: string
  disabled?: boolean
  error?: Error | null
  pending?: boolean
  children: React.ReactNode
}) {
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>{title}</h3>
        {children}
        {error && <div className="error-banner">{error.message}</div>}
        <div className="toolbar">
          <button className="btn primary" disabled={disabled || pending} onClick={onSubmit}>
            {submitLabel}
          </button>
          <button className="btn" onClick={onClose}>
            Cancelar
          </button>
        </div>
      </div>
    </div>
  )
}

// ======================= Productos =======================

function ProductsTab() {
  const queryClient = useQueryClient()
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: CatalogApi.categories })
  const { data: products } = useQuery({ queryKey: ['products-all'], queryFn: () => CatalogApi.products(false) })
  const { data: groups } = useQuery({ queryKey: ['modifier-groups'], queryFn: CatalogApi.modifierGroups })

  const [selected, setSelected] = useState<number | 'all'>('all')
  const [search, setSearch] = useState('')
  const [productForm, setProductForm] = useState<{ product: Product | null } | null>(null)
  const [categoryForm, setCategoryForm] = useState<{ category: Category | null } | null>(null)

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['products-all'] })
    queryClient.invalidateQueries({ queryKey: ['products'] })
    queryClient.invalidateQueries({ queryKey: ['categories'] })
  }

  const toggle = useMutation({
    mutationFn: (p: Product) =>
      CatalogApi.updateProduct(p.id, {
        category_id: p.category_id,
        name: p.name,
        price: p.price,
        active: !p.active,
        group_ids: p.modifier_groups.map((g) => g.id),
      }),
    onSuccess: refresh,
  })
  const removeProduct = useMutation({ mutationFn: (id: number) => CatalogApi.deleteProduct(id), onSuccess: refresh })
  const removeCategory = useMutation({
    mutationFn: (id: number) => CatalogApi.deleteCategory(id),
    onSuccess: () => {
      setSelected('all')
      refresh()
    },
  })

  const shown = (products ?? [])
    .filter((p) => selected === 'all' || p.category_id === selected)
    .filter((p) => p.name.toLowerCase().includes(search.toLowerCase()))
  const currentCategory = categories?.find((c) => c.id === selected)
  const err = (toggle.error ?? removeProduct.error ?? removeCategory.error) as Error | null

  return (
    <div className="catalog-layout">
      <aside className="card category-list">
        <div className="section-title" style={{ marginTop: 0 }}>
          Categorías
        </div>
        <button className={`cat-item ${selected === 'all' ? 'active' : ''}`} onClick={() => setSelected('all')}>
          Todas <span className="muted">{products?.length ?? 0}</span>
        </button>
        {(categories ?? []).map((c) => (
          <button key={c.id} className={`cat-item ${selected === c.id ? 'active' : ''}`} onClick={() => setSelected(c.id)}>
            {c.name} <span className="muted">{(products ?? []).filter((p) => p.category_id === c.id).length}</span>
          </button>
        ))}
        <button className="btn small" style={{ marginTop: '0.5rem' }} onClick={() => setCategoryForm({ category: null })}>
          + Nueva categoría
        </button>
        {currentCategory && (
          <div className="manage-actions" style={{ marginTop: '0.5rem' }}>
            <button className="btn small" onClick={() => setCategoryForm({ category: currentCategory })}>
              Renombrar
            </button>
            <button
              className="btn small danger"
              onClick={() => confirm(`¿Eliminar la categoría "${currentCategory.name}"? Debe estar vacía.`) && removeCategory.mutate(currentCategory.id)}
            >
              Eliminar
            </button>
          </div>
        )}
      </aside>

      <section>
        <div className="toolbar">
          <input className="search" placeholder="Buscar producto…" value={search} onChange={(e) => setSearch(e.target.value)} />
          <button className="btn primary" disabled={(categories ?? []).length === 0} onClick={() => setProductForm({ product: null })}>
            + Nuevo producto
          </button>
        </div>
        {(categories ?? []).length === 0 && <p className="muted">Empieza creando una categoría (ej. Tacos, Bebidas).</p>}
        {err && <div className="error-banner">{err.message}</div>}

        <div className="cards-grid">
          {shown.map((p) => (
            <div className={`card product-card ${p.active ? '' : 'inactive'}`} key={p.id}>
              <div className="order-card-head">
                <strong>{p.name}</strong>
                <span className="price">{formatCurrency(p.price)}</span>
              </div>
              <div className="muted">{categories?.find((c) => c.id === p.category_id)?.name}</div>
              <div className="chip-row">
                {p.modifier_groups.map((g) => (
                  <span key={g.id} className="status-badge">
                    {g.name}
                  </span>
                ))}
              </div>
              <div className="manage-actions">
                <button className="btn small" onClick={() => setProductForm({ product: p })}>
                  Editar
                </button>
                <button className="btn small" disabled={toggle.isPending} onClick={() => toggle.mutate(p)}>
                  {p.active ? 'Ocultar del menú' : 'Mostrar en menú'}
                </button>
                <button
                  className="btn small danger"
                  onClick={() =>
                    confirm(`¿Eliminar "${p.name}"? Si ya se vendió, mejor ocúltalo del menú.`) && removeProduct.mutate(p.id)
                  }
                >
                  Eliminar
                </button>
              </div>
            </div>
          ))}
        </div>
        {shown.length === 0 && (categories ?? []).length > 0 && <p className="muted">No hay productos aquí todavía.</p>}
      </section>

      {productForm && (
        <ProductFormModal
          product={productForm.product}
          categories={categories ?? []}
          groups={groups ?? []}
          defaultCategory={selected === 'all' ? undefined : selected}
          onClose={() => setProductForm(null)}
          onSaved={refresh}
        />
      )}
      {categoryForm && (
        <CategoryFormModal
          category={categoryForm.category}
          nextOrder={(categories?.length ?? 0) + 1}
          onClose={() => setCategoryForm(null)}
          onSaved={refresh}
        />
      )}
    </div>
  )
}

function CategoryFormModal({
  category,
  nextOrder,
  onClose,
  onSaved,
}: {
  category: Category | null
  nextOrder: number
  onClose: () => void
  onSaved: () => void
}) {
  const [name, setName] = useState(category?.name ?? '')
  const save = useMutation({
    mutationFn: () =>
      category
        ? CatalogApi.updateCategory(category.id, { name, sort_order: category.sort_order })
        : CatalogApi.createCategory({ name, sort_order: nextOrder }),
    onSuccess: () => {
      onSaved()
      onClose()
    },
  })
  return (
    <Dialog
      title={category ? 'Renombrar categoría' : 'Nueva categoría'}
      onClose={onClose}
      onSubmit={() => save.mutate()}
      disabled={!name}
      error={save.error as Error | null}
      pending={save.isPending}
    >
      <Field label="Nombre">
        <input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
    </Dialog>
  )
}

function ProductFormModal({
  product,
  categories,
  groups,
  defaultCategory,
  onClose,
  onSaved,
}: {
  product: Product | null
  categories: Category[]
  groups: ModifierGroup[]
  defaultCategory?: number
  onClose: () => void
  onSaved: () => void
}) {
  const [name, setName] = useState(product?.name ?? '')
  const [price, setPrice] = useState(product ? String(product.price) : '')
  const [categoryId, setCategoryId] = useState(product?.category_id ?? defaultCategory ?? categories[0]?.id ?? 0)
  const [active, setActive] = useState(product?.active ?? true)
  const [groupIds, setGroupIds] = useState<number[]>(product?.modifier_groups.map((g) => g.id) ?? [])

  const save = useMutation({
    mutationFn: () => {
      const input = { category_id: categoryId, name, price: Number(price), active, group_ids: groupIds }
      return product ? CatalogApi.updateProduct(product.id, input) : CatalogApi.createProduct(input)
    },
    onSuccess: () => {
      onSaved()
      onClose()
    },
  })

  return (
    <Dialog
      title={product ? `Editar ${product.name}` : 'Nuevo producto'}
      onClose={onClose}
      onSubmit={() => save.mutate()}
      disabled={!name || price === '' || !categoryId}
      error={save.error as Error | null}
      pending={save.isPending}
    >
      <Field label="Nombre">
        <input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
      <div className="form-row">
        <Field label="Categoría">
          <select value={categoryId} onChange={(e) => setCategoryId(Number(e.target.value))}>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Precio">
          <input type="number" value={price} onChange={(e) => setPrice(e.target.value)} />
        </Field>
      </div>
      <div className="section-title" style={{ marginTop: 0 }}>
        Opciones que se le ofrecen al cliente
      </div>
      {groups.length === 0 && <p className="muted">Aún no hay modificadores; créalos en la pestaña “Modificadores”.</p>}
      <div className="chip-row" style={{ marginBottom: '0.75rem' }}>
        {groups.map((g) => (
          <button
            key={g.id}
            type="button"
            className={`chip ${groupIds.includes(g.id) ? 'selected' : ''}`}
            onClick={() => setGroupIds((ids) => (ids.includes(g.id) ? ids.filter((i) => i !== g.id) : [...ids, g.id]))}
          >
            {g.name}
          </button>
        ))}
      </div>
      <label className="check" style={{ marginBottom: '0.75rem' }}>
        <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />
        Visible en el menú
      </label>
    </Dialog>
  )
}

// ======================= Modificadores =======================

function ModifiersTab() {
  const queryClient = useQueryClient()
  const { data: groups } = useQuery({ queryKey: ['modifier-groups'], queryFn: CatalogApi.modifierGroups })
  const [groupForm, setGroupForm] = useState<{ group: ModifierGroup | null } | null>(null)
  const [optionForm, setOptionForm] = useState<{ group: ModifierGroup; option: { id: number; name: string; price_delta: number } | null } | null>(null)

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['modifier-groups'] })
    queryClient.invalidateQueries({ queryKey: ['products-all'] })
    queryClient.invalidateQueries({ queryKey: ['products'] })
  }
  const removeGroup = useMutation({ mutationFn: (id: number) => CatalogApi.deleteModifierGroup(id), onSuccess: refresh })
  const removeOption = useMutation({
    mutationFn: ({ g, o }: { g: number; o: number }) => CatalogApi.deleteModifierOption(g, o),
    onSuccess: refresh,
  })
  const err = (removeGroup.error ?? removeOption.error) as Error | null

  return (
    <div>
      <div className="toolbar">
        <button className="btn primary" onClick={() => setGroupForm({ group: null })}>
          + Nuevo grupo
        </button>
        <span className="muted">Ej.: “Guisado” (pastor, suadero…), “Extras” (queso +$10), “Término”.</span>
      </div>
      {err && <div className="error-banner">{err.message}</div>}

      <div className="cards-grid">
        {(groups ?? []).map((g) => (
          <div className="card" key={g.id}>
            <div className="order-card-head">
              <strong>{g.name}</strong>
              <span className="muted">
                {g.required ? 'Obligatorio' : 'Opcional'} · {g.multiple ? 'varias' : 'una'}
              </span>
            </div>
            {g.options.length === 0 && <p className="muted">Sin opciones todavía.</p>}
            {g.options.map((o) => (
              <div className="manage-row" key={o.id}>
                <div className="manage-info">
                  <span>{o.name}</span>
                  <span className="muted">{o.price_delta ? `+ ${formatCurrency(o.price_delta)}` : 'Sin costo extra'}</span>
                </div>
                <div className="manage-actions">
                  <button className="btn small" onClick={() => setOptionForm({ group: g, option: o })}>
                    Editar
                  </button>
                  <button
                    className="btn small danger"
                    onClick={() => confirm(`¿Eliminar la opción "${o.name}"?`) && removeOption.mutate({ g: g.id, o: o.id })}
                  >
                    Quitar
                  </button>
                </div>
              </div>
            ))}
            <div className="manage-actions" style={{ marginTop: '0.5rem' }}>
              <button className="btn small primary" onClick={() => setOptionForm({ group: g, option: null })}>
                + Opción
              </button>
              <button className="btn small" onClick={() => setGroupForm({ group: g })}>
                Editar grupo
              </button>
              <button
                className="btn small danger"
                onClick={() => confirm(`¿Eliminar el grupo "${g.name}" con todas sus opciones? Se quita de los productos que lo usan.`) && removeGroup.mutate(g.id)}
              >
                Eliminar grupo
              </button>
            </div>
          </div>
        ))}
      </div>

      {groupForm && <GroupFormModal group={groupForm.group} onClose={() => setGroupForm(null)} onSaved={refresh} />}
      {optionForm && <OptionFormModal {...optionForm} onClose={() => setOptionForm(null)} onSaved={refresh} />}
    </div>
  )
}

function GroupFormModal({ group, onClose, onSaved }: { group: ModifierGroup | null; onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState(group?.name ?? '')
  const [required, setRequired] = useState(group?.required ?? false)
  const [multiple, setMultiple] = useState(group?.multiple ?? false)
  const save = useMutation({
    mutationFn: () => {
      const input = { name, required, multiple }
      return group ? CatalogApi.updateModifierGroup(group.id, input) : CatalogApi.createModifierGroup(input)
    },
    onSuccess: () => {
      onSaved()
      onClose()
    },
  })
  return (
    <Dialog
      title={group ? 'Editar grupo' : 'Nuevo grupo de modificadores'}
      onClose={onClose}
      onSubmit={() => save.mutate()}
      disabled={!name}
      error={save.error as Error | null}
      pending={save.isPending}
    >
      <Field label="Nombre del grupo">
        <input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
      <label className="check" style={{ marginBottom: '0.5rem' }}>
        <input type="checkbox" checked={required} onChange={(e) => setRequired(e.target.checked)} />
        Obligatorio (hay que elegir al menos una opción)
      </label>
      <label className="check" style={{ marginBottom: '0.75rem' }}>
        <input type="checkbox" checked={multiple} onChange={(e) => setMultiple(e.target.checked)} />
        Se pueden elegir varias opciones
      </label>
    </Dialog>
  )
}

function OptionFormModal({
  group,
  option,
  onClose,
  onSaved,
}: {
  group: ModifierGroup
  option: { id: number; name: string; price_delta: number } | null
  onClose: () => void
  onSaved: () => void
}) {
  const [name, setName] = useState(option?.name ?? '')
  const [delta, setDelta] = useState(String(option?.price_delta ?? 0))
  const save = useMutation({
    mutationFn: () => {
      const input = { name, price_delta: Number(delta) }
      return option ? CatalogApi.updateModifierOption(group.id, option.id, input) : CatalogApi.addModifierOption(group.id, input)
    },
    onSuccess: () => {
      onSaved()
      onClose()
    },
  })
  return (
    <Dialog
      title={option ? `Editar opción de ${group.name}` : `Nueva opción de ${group.name}`}
      onClose={onClose}
      onSubmit={() => save.mutate()}
      disabled={!name}
      error={save.error as Error | null}
      pending={save.isPending}
    >
      <Field label="Nombre">
        <input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
      <Field label="Costo extra (0 si no cuesta más)">
        <input type="number" value={delta} onChange={(e) => setDelta(e.target.value)} />
      </Field>
    </Dialog>
  )
}
