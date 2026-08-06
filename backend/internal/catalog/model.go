package catalog

type Category struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type ModifierOption struct {
	ID         int     `json:"id"`
	GroupID    int     `json:"group_id"`
	Name       string  `json:"name"`
	PriceDelta float64 `json:"price_delta"`
}

type ModifierGroup struct {
	ID       int              `json:"id"`
	Name     string           `json:"name"`
	Required bool             `json:"required"`
	Multiple bool             `json:"multiple"`
	Options  []ModifierOption `json:"options"`
}

type Product struct {
	ID             int             `json:"id"`
	CategoryID     int             `json:"category_id"`
	Name           string          `json:"name"`
	Price          float64         `json:"price"`
	Active         bool            `json:"active"`
	ModifierGroups []ModifierGroup `json:"modifier_groups"`
}

type CategoryInput struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type ProductInput struct {
	CategoryID int     `json:"category_id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Active     bool    `json:"active"`
	GroupIDs   []int   `json:"group_ids"`
}

type ModifierGroupInput struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Multiple bool   `json:"multiple"`
}

type ModifierOptionInput struct {
	Name       string  `json:"name"`
	PriceDelta float64 `json:"price_delta"`
}
