package cashcuts

import (
	"errors"
	"fmt"
)

var ErrValidation = errors.New("datos inválidos")

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// OpenShiftIDForCashier implementa accounts.ShiftFinder.
func (s *Service) OpenShiftIDForCashier(cashierID int) (*int, error) {
	return s.store.OpenShiftIDForCashier(cashierID)
}

func (s *Service) OpenShift(requesterID int, in OpenShiftInput) (Shift, error) {
	cashierID := requesterID
	if in.CashierID != 0 {
		cashierID = in.CashierID
	}
	open, err := s.store.HasOpenShift(cashierID)
	if err != nil {
		return Shift{}, err
	}
	if open {
		return Shift{}, ErrAlreadyOpen
	}
	if in.OpeningFund < 0 {
		return Shift{}, fmt.Errorf("%w: el fondo inicial no puede ser negativo", ErrValidation)
	}
	return s.store.OpenShift(cashierID, in.OpeningFund)
}

func (s *Service) GetShift(id int) (Shift, error) { return s.store.GetShift(id) }

func (s *Service) ListShifts(status string) ([]Shift, error) { return s.store.ListShifts(status) }

// CloseShift cierra el turno y genera automáticamente el corte Z final.
func (s *Service) CloseShift(id int, in CloseShiftInput) (Shift, CashCut, error) {
	shift, err := s.store.GetShift(id)
	if err != nil {
		return Shift{}, CashCut{}, err
	}
	if shift.Status != "abierto" {
		return Shift{}, CashCut{}, fmt.Errorf("%w: el turno ya está cerrado", ErrValidation)
	}
	closed, err := s.store.CloseShift(id, in.CountedCash)
	if err != nil {
		return Shift{}, CashCut{}, err
	}
	cut, err := s.store.CreateCut(id, "z", &in.CountedCash, shift.CashierID)
	if err != nil {
		return Shift{}, CashCut{}, err
	}
	return closed, cut, nil
}

// GenerateXCut produce un corte parcial de consulta sin cerrar el turno.
func (s *Service) GenerateXCut(shiftID int, in CutInput, createdBy int) (CashCut, error) {
	shift, err := s.store.GetShift(shiftID)
	if err != nil {
		return CashCut{}, err
	}
	if shift.Status != "abierto" {
		return CashCut{}, fmt.Errorf("%w: el turno ya está cerrado", ErrValidation)
	}
	return s.store.CreateCut(shiftID, "x", in.CountedCash, createdBy)
}

func (s *Service) GetCut(id int) (CashCut, error) { return s.store.GetCut(id) }

func (s *Service) ListCutsByShift(shiftID int) ([]CashCut, error) {
	return s.store.ListCutsByShift(shiftID)
}

func (s *Service) ListCuts(from, to, cutType string) ([]CashCut, error) {
	if from == "" {
		from = "0000-01-01"
	}
	if to == "" {
		to = "9999-12-31"
	}
	return s.store.ListCuts(from, to, cutType)
}
