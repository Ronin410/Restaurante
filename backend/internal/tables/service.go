package tables

import (
	"errors"
	"fmt"
	"time"
)

var ErrValidation = errors.New("datos inválidos")

var validStatuses = map[string]bool{"libre": true, "ocupada": true, "cuenta_solicitada": true}

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListZones() ([]Zone, error) { return s.store.ListZones() }

func (s *Service) CreateZone(in ZoneInput) (Zone, error) {
	if in.Name == "" {
		return Zone{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	return s.store.CreateZone(in)
}

func (s *Service) UpdateZone(id int, in ZoneInput) (Zone, error) {
	if in.Name == "" {
		return Zone{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	return s.store.UpdateZone(id, in)
}

func (s *Service) DeleteZone(id int) error {
	return s.store.DeleteZone(id)
}

func (s *Service) ListTables(includeInactive bool) ([]Table, error) {
	list, err := s.store.ListTables(includeInactive)
	if err != nil {
		return nil, err
	}
	for i := range list {
		withOccupancy(&list[i])
	}
	return list, nil
}

func (s *Service) GetTable(id int) (Table, error) {
	t, err := s.store.GetTable(id)
	if err != nil {
		return t, err
	}
	withOccupancy(&t)
	return t, nil
}

func withOccupancy(t *Table) {
	if t.Status == "ocupada" && t.OpenedAt != nil {
		if opened, err := time.Parse("2006-01-02 15:04:05", *t.OpenedAt); err == nil {
			mins := int(time.Since(opened).Minutes())
			t.OccupiedMinutes = &mins
		}
	}
}

func (s *Service) CreateTable(in TableInput) (Table, error) {
	if in.Number == "" || in.ZoneID == 0 {
		return Table{}, fmt.Errorf("%w: número y zona obligatorios", ErrValidation)
	}
	if in.Capacity <= 0 {
		in.Capacity = 4
	}
	return s.store.CreateTable(in)
}

func (s *Service) UpdateTable(id int, in TableInput) (Table, error) {
	if in.Number == "" || in.ZoneID == 0 {
		return Table{}, fmt.Errorf("%w: número y zona obligatorios", ErrValidation)
	}
	return s.store.UpdateTable(id, in)
}

func (s *Service) Open(id, waiterID int) (Table, error) {
	t, err := s.store.GetTable(id)
	if err != nil {
		return Table{}, err
	}
	if !t.Active {
		return Table{}, fmt.Errorf("%w: la mesa fue eliminada; reactívala primero", ErrValidation)
	}
	if t.Status != "libre" {
		return Table{}, fmt.Errorf("%w: la mesa no está libre", ErrValidation)
	}
	return s.store.Open(id, waiterID)
}

func (s *Service) SetStatus(id int, status string) (Table, error) {
	if !validStatuses[status] {
		return Table{}, fmt.Errorf("%w: estatus inválido", ErrValidation)
	}
	return s.store.SetStatus(id, status)
}

// Release deja la mesa libre (cierra su sesión de ocupación para el KPI de
// rotación). Lo usa el cierre de cuentas; no existe estado de limpieza.
func (s *Service) Release(id int) error {
	_, err := s.store.SetStatus(id, "libre")
	return err
}

func (s *Service) AssignWaiter(id, waiterID int) (Table, error) {
	return s.store.AssignWaiter(id, waiterID)
}

// Join une la mesa withTableID al grupo de tableID (mesas grandes).
func (s *Service) Join(tableID, withTableID int) (Table, error) {
	if tableID == withTableID {
		return Table{}, fmt.Errorf("%w: no se puede unir una mesa consigo misma", ErrValidation)
	}
	return s.store.Join(tableID, withTableID)
}

func (s *Service) Unjoin(tableID int) (Table, error) {
	return s.store.Unjoin(tableID)
}

func (s *Service) Transfer(tableID int, toWaiterID, authorizedBy int) (TransferRecord, error) {
	current, err := s.store.GetTable(tableID)
	if err != nil {
		return TransferRecord{}, err
	}
	return s.store.Transfer(tableID, current.CurrentWaiterID, toWaiterID, authorizedBy)
}

// Deactivate "elimina" una mesa. No se permite si está ocupada o con
// cuenta solicitada, para no perder de vista una comanda en curso; hay que
// liberarla (o cerrar su cuenta) antes de eliminarla.
func (s *Service) Deactivate(id int) (Table, error) {
	t, err := s.store.GetTable(id)
	if err != nil {
		return Table{}, err
	}
	if t.Status == "ocupada" || t.Status == "cuenta_solicitada" {
		return Table{}, fmt.Errorf("%w: libera la mesa antes de eliminarla", ErrValidation)
	}
	return s.store.Deactivate(id)
}

func (s *Service) Reactivate(id int) (Table, error) {
	return s.store.Reactivate(id)
}
