package tables

type Zone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Table struct {
	ID                int     `json:"id"`
	ZoneID            int     `json:"zone_id"`
	ZoneName          string  `json:"zone_name,omitempty"`
	Number            string  `json:"number"`
	Capacity          int     `json:"capacity"`
	Status            string  `json:"status"` // libre | ocupada | cuenta_solicitada | limpieza
	Active            bool    `json:"active"`
	CurrentWaiterID   *int    `json:"current_waiter_id"`
	CurrentWaiterName *string `json:"current_waiter_name,omitempty"`
	OpenedAt          *string `json:"opened_at"`
	OccupiedMinutes   *int    `json:"occupied_minutes,omitempty"`
	JoinedWith        *int    `json:"joined_with"`
}

type ZoneInput struct {
	Name string `json:"name"`
}

type TableInput struct {
	ZoneID   int    `json:"zone_id"`
	Number   string `json:"number"`
	Capacity int    `json:"capacity"`
}

type TransferInput struct {
	ToWaiterID int `json:"to_waiter_id"`
}

type TransferRecord struct {
	ID           int    `json:"id"`
	TableID      int    `json:"table_id"`
	FromWaiterID *int   `json:"from_waiter_id"`
	ToWaiterID   int    `json:"to_waiter_id"`
	AuthorizedBy int    `json:"authorized_by"`
	CreatedAt    string `json:"created_at"`
}
