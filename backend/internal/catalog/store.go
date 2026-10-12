package catalog

import (
	"database/sql"

	"errors"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

var ErrInUse = errors.New("no se puede eliminar porque ya se usa en ventas o en otros elementos; mejor desactívalo")
var ErrNotFound = errors.New("elemento de catálogo no encontrado")

type Store struct {
	db *dbx.DB
}

func NewStore(db *dbx.DB) *Store {
	return &Store{db: db}
}

// ---- Categorías ----

func (s *Store) ListCategories() ([]Category, error) {
	rows, err := s.db.Query(`SELECT id, name, sort_order FROM categories ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCategory(in CategoryInput) (Category, error) {
	var id int
	err := s.db.QueryRow(`INSERT INTO categories (name, sort_order) VALUES (?, ?) RETURNING id`, in.Name, in.SortOrder).Scan(&id)
	if err != nil {
		return Category{}, err
	}
	return Category{ID: id, Name: in.Name, SortOrder: in.SortOrder}, nil
}

func (s *Store) UpdateCategory(id int, in CategoryInput) (Category, error) {
	_, err := s.db.Exec(`UPDATE categories SET name = ?, sort_order = ? WHERE id = ?`, in.Name, in.SortOrder, id)
	if err != nil {
		return Category{}, err
	}
	return Category{ID: id, Name: in.Name, SortOrder: in.SortOrder}, nil
}

func (s *Store) DeleteCategory(id int) error {
	_, err := s.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return ErrInUse
	}
	return nil
}

// ---- Modificadores ----

func (s *Store) ListModifierGroups() ([]ModifierGroup, error) {
	rows, err := s.db.Query(`SELECT id, name, required, multiple FROM modifier_groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []ModifierGroup
	for rows.Next() {
		var g ModifierGroup
		var req, mult int
		if err := rows.Scan(&g.ID, &g.Name, &req, &mult); err != nil {
			return nil, err
		}
		g.Required, g.Multiple = req == 1, mult == 1
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range groups {
		opts, err := s.listOptions(groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Options = opts
	}
	return groups, nil
}

func (s *Store) listOptions(groupID int) ([]ModifierOption, error) {
	rows, err := s.db.Query(`SELECT id, group_id, name, price_delta FROM modifier_options WHERE group_id = ? ORDER BY id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModifierOption
	for rows.Next() {
		var o ModifierOption
		if err := rows.Scan(&o.ID, &o.GroupID, &o.Name, &o.PriceDelta); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) CreateModifierGroup(in ModifierGroupInput) (ModifierGroup, error) {
	req, mult := 0, 0
	if in.Required {
		req = 1
	}
	if in.Multiple {
		mult = 1
	}
	var id int
	err := s.db.QueryRow(`INSERT INTO modifier_groups (name, required, multiple) VALUES (?, ?, ?) RETURNING id`, in.Name, req, mult).Scan(&id)
	if err != nil {
		return ModifierGroup{}, err
	}
	return ModifierGroup{ID: id, Name: in.Name, Required: in.Required, Multiple: in.Multiple}, nil
}

func (s *Store) AddModifierOption(groupID int, in ModifierOptionInput) (ModifierOption, error) {
	var id int
	err := s.db.QueryRow(`INSERT INTO modifier_options (group_id, name, price_delta) VALUES (?, ?, ?) RETURNING id`, groupID, in.Name, in.PriceDelta).Scan(&id)
	if err != nil {
		return ModifierOption{}, err
	}
	return ModifierOption{ID: id, GroupID: groupID, Name: in.Name, PriceDelta: in.PriceDelta}, nil
}

// ---- Productos ----

func (s *Store) ListProducts(activeOnly bool) ([]Product, error) {
	query := `SELECT id, category_id, name, price, active FROM products`
	if activeOnly {
		query += ` WHERE active = 1`
	}
	query += ` ORDER BY name`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range products {
		groups, err := s.productModifierGroups(products[i].ID)
		if err != nil {
			return nil, err
		}
		products[i].ModifierGroups = groups
	}
	return products, nil
}

func scanProduct(row interface {
	Scan(dest ...interface{}) error
}) (Product, error) {
	var p Product
	var active int
	err := row.Scan(&p.ID, &p.CategoryID, &p.Name, &p.Price, &active)
	p.Active = active == 1
	return p, err
}

func (s *Store) GetProduct(id int) (Product, error) {
	row := s.db.QueryRow(`SELECT id, category_id, name, price, active FROM products WHERE id = ?`, id)
	p, err := scanProduct(row)
	if err == sql.ErrNoRows {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	groups, err := s.productModifierGroups(p.ID)
	if err != nil {
		return p, err
	}
	p.ModifierGroups = groups
	return p, nil
}

func (s *Store) productModifierGroups(productID int) ([]ModifierGroup, error) {
	rows, err := s.db.Query(`
		SELECT mg.id, mg.name, mg.required, mg.multiple
		FROM product_modifier_groups pmg
		JOIN modifier_groups mg ON mg.id = pmg.group_id
		WHERE pmg.product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []ModifierGroup
	for rows.Next() {
		var g ModifierGroup
		var req, mult int
		if err := rows.Scan(&g.ID, &g.Name, &req, &mult); err != nil {
			return nil, err
		}
		g.Required, g.Multiple = req == 1, mult == 1
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range groups {
		opts, err := s.listOptions(groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Options = opts
	}
	return groups, nil
}

func (s *Store) CreateProduct(in ProductInput) (Product, error) {
	activeInt := 1
	if !in.Active {
		activeInt = 0
	}
	var id int
	err := s.db.QueryRow(`INSERT INTO products (category_id, name, price, active) VALUES (?, ?, ?, ?) RETURNING id`,
		in.CategoryID, in.Name, in.Price, activeInt).Scan(&id)
	if err != nil {
		return Product{}, err
	}
	if err := s.setProductGroups(id, in.GroupIDs); err != nil {
		return Product{}, err
	}
	return s.GetProduct(id)
}

func (s *Store) UpdateProduct(id int, in ProductInput) (Product, error) {
	activeInt := 1
	if !in.Active {
		activeInt = 0
	}
	_, err := s.db.Exec(`UPDATE products SET category_id = ?, name = ?, price = ?, active = ? WHERE id = ?`,
		in.CategoryID, in.Name, in.Price, activeInt, id)
	if err != nil {
		return Product{}, err
	}
	if err := s.setProductGroups(id, in.GroupIDs); err != nil {
		return Product{}, err
	}
	return s.GetProduct(id)
}

func (s *Store) setProductGroups(productID int, groupIDs []int) error {
	if _, err := s.db.Exec(`DELETE FROM product_modifier_groups WHERE product_id = ?`, productID); err != nil {
		return err
	}
	for _, gid := range groupIDs {
		if _, err := s.db.Exec(`INSERT INTO product_modifier_groups (product_id, group_id) VALUES (?, ?)`, productID, gid); err != nil {
			return err
		}
	}
	return nil
}

// ---- Edición y baja de modificadores y productos ----

func (s *Store) UpdateModifierGroup(id int, in ModifierGroupInput) (ModifierGroup, error) {
	res, err := s.db.Exec(`UPDATE modifier_groups SET name = ?, required = ?, multiple = ? WHERE id = ?`,
		in.Name, boolInt(in.Required), boolInt(in.Multiple), id)
	if err != nil {
		return ModifierGroup{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ModifierGroup{}, ErrNotFound
	}
	groups, err := s.ListModifierGroups()
	if err != nil {
		return ModifierGroup{}, err
	}
	for _, g := range groups {
		if g.ID == id {
			return g, nil
		}
	}
	return ModifierGroup{}, ErrNotFound
}

// DeleteModifierGroup borra el grupo, sus opciones y su asignación a productos.
func (s *Store) DeleteModifierGroup(id int) error {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM product_modifier_groups WHERE group_id = $1`,
		`DELETE FROM modifier_options WHERE group_id = $1`,
		`DELETE FROM modifier_groups WHERE id = $1`,
	} {
		if _, err := tx.Exec(q, id); err != nil {
			return ErrInUse
		}
	}
	return tx.Commit()
}

func (s *Store) UpdateModifierOption(groupID, optionID int, in ModifierOptionInput) (ModifierOption, error) {
	res, err := s.db.Exec(`UPDATE modifier_options SET name = ?, price_delta = ? WHERE id = ? AND group_id = ?`,
		in.Name, in.PriceDelta, optionID, groupID)
	if err != nil {
		return ModifierOption{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ModifierOption{}, ErrNotFound
	}
	return ModifierOption{ID: optionID, GroupID: groupID, Name: in.Name, PriceDelta: in.PriceDelta}, nil
}

func (s *Store) DeleteModifierOption(groupID, optionID int) error {
	res, err := s.db.Exec(`DELETE FROM modifier_options WHERE id = ? AND group_id = ?`, optionID, groupID)
	if err != nil {
		return ErrInUse
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteProduct borra el producto si nunca se vendió; si ya tiene ventas la
// base lo impide (ErrInUse) y lo correcto es desactivarlo.
func (s *Store) DeleteProduct(id int) error {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM product_modifier_groups WHERE product_id = $1`, id); err != nil {
		return ErrInUse
	}
	res, err := tx.Exec(`DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return ErrInUse
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
