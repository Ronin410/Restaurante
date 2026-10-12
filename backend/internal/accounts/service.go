package accounts

import (
	"errors"
	"fmt"
	"sync"
)

var ErrValidation = errors.New("datos inválidos")

var validMethods = map[string]bool{
	"efectivo": true, "tarjeta_debito": true, "tarjeta_credito": true, "transferencia": true, "vale_despensa": true,
}
var validDiscountKinds = map[string]bool{"descuento": true, "cortesia": true}

const balanceTolerance = 0.01

// ShiftFinder desacopla accounts del módulo cashcuts: al registrar un pago
// en efectivo/tarjeta se asocia al turno de caja abierto del cajero (si
// existe) para que el corte de caja pueda desglosarlo correctamente.
type ShiftFinder interface {
	OpenShiftIDForCashier(cashierID int) (*int, error)
}

// TableReleaser desacopla accounts del módulo tables: al cerrar la última
// cuenta pendiente de una mesa, la mesa se libera sola (vuelve a "libre").
// El sistema no maneja ningún estado de limpieza: la mesa queda disponible
// de inmediato y el personal la prepara por su cuenta.
type TableReleaser interface {
	Release(tableID int) error
	// RequestBill marca la mesa como "cuenta solicitada" al generar la cuenta.
	RequestBill(tableID int) error
}

type Service struct {
	// genMu serializa la generación de cuentas: sin esto, dos peticiones casi
	// simultáneas (doble toque) podrían crear dos cuentas con las mismas partidas.
	genMu    sync.Mutex
	store    *Store
	shifts   ShiftFinder
	releaser TableReleaser
}

func NewService(store *Store, shifts ShiftFinder, releaser TableReleaser) *Service {
	return &Service{store: store, shifts: shifts, releaser: releaser}
}

// GenerateOrderAccount genera la cuenta de un pedido sin mesa (llevar/domicilio).
func (s *Service) GenerateOrderAccount(orderID int) (Account, error) {
	s.genMu.Lock()
	defer s.genMu.Unlock()
	acc, err := s.store.GenerateAccountForOrder(orderID)
	if errors.Is(err, ErrNoItems) {
		if open, oerr := s.store.OpenAccountForOrder(orderID); oerr == nil && open != 0 {
			return s.store.GetAccount(open)
		}
	}
	return acc, err
}

func (s *Service) GenerateAccount(tableID int) (Account, error) {
	if tableID == 0 {
		return Account{}, fmt.Errorf("%w: mesa o pedido obligatorio", ErrValidation)
	}
	s.genMu.Lock()
	defer s.genMu.Unlock()
	acc, err := s.store.GenerateAccount(tableID)
	if errors.Is(err, ErrNoItems) {
		// Si ya hay una cuenta abierta (ya se generó/imprimió pero falta
		// cobrarla), se regresa esa en lugar de un error: así se puede
		// retomar el cobro y cerrarla, que es lo que libera la mesa.
		if open, oerr := s.store.OpenAccountForTable(tableID); oerr == nil && open != 0 {
			return s.store.GetAccount(open)
		}
		return Account{}, err
	}
	if err != nil {
		return Account{}, err
	}
	if s.releaser != nil {
		if rerr := s.releaser.RequestBill(tableID); rerr != nil {
			return Account{}, rerr
		}
	}
	return acc, nil
}

func (s *Service) SplitByProduct(in SplitByProductInput) ([]Account, error) {
	if in.TableID == 0 || len(in.Groups) == 0 {
		return nil, fmt.Errorf("%w: mesa y grupos de productos son obligatorios", ErrValidation)
	}
	return s.store.SplitByProduct(in.TableID, in.Groups)
}

func (s *Service) GetAccount(id int) (Account, error) {
	return s.store.GetAccount(id)
}

func (s *Service) ListByTable(tableID int) ([]Account, error) {
	return s.store.ListByTable(tableID)
}

// AddDiscount aplica un descuento o cortesía. El endpoint debe restringirse
// a roles supervisores (gerente/admin); aquí se valida el motivo obligatorio
// como exige el control de mermas/fraude.
func (s *Service) AddDiscount(accountID int, in DiscountInput, authorizedBy int) (Account, error) {
	if !validDiscountKinds[in.Kind] {
		return Account{}, fmt.Errorf("%w: tipo de descuento inválido", ErrValidation)
	}
	if in.Reason == "" {
		return Account{}, fmt.Errorf("%w: el motivo es obligatorio", ErrValidation)
	}
	account, err := s.store.GetAccount(accountID)
	if err != nil {
		return Account{}, err
	}
	if account.Status != "abierta" {
		return Account{}, fmt.Errorf("%w: la cuenta ya está cerrada", ErrValidation)
	}
	amount := in.Amount
	if in.Percent != nil && *in.Percent > 0 {
		amount = account.Subtotal * (*in.Percent / 100)
	}
	if amount <= 0 {
		return Account{}, fmt.Errorf("%w: el monto del descuento debe ser mayor a cero", ErrValidation)
	}
	in.Amount = amount
	return s.store.AddDiscount(accountID, in, authorizedBy)
}

// AddPayment registra un pago (posiblemente parcial, para permitir cuentas
// divididas por persona o por monto libre: cada persona paga su parte con
// llamadas independientes a este método hasta saldar el total).
func (s *Service) AddPayment(accountID int, in PaymentInput, createdBy int) (Account, error) {
	if !validMethods[in.Method] {
		return Account{}, fmt.Errorf("%w: método de pago inválido", ErrValidation)
	}
	if in.Amount <= 0 {
		return Account{}, fmt.Errorf("%w: el monto debe ser mayor a cero", ErrValidation)
	}
	account, err := s.store.GetAccount(accountID)
	if err != nil {
		return Account{}, err
	}
	if account.Status != "abierta" {
		return Account{}, fmt.Errorf("%w: la cuenta ya está cerrada", ErrValidation)
	}
	if in.Amount > account.Balance+balanceTolerance {
		return Account{}, fmt.Errorf("%w: el pago excede el saldo pendiente (%.2f)", ErrValidation, account.Balance)
	}

	var shiftID *int
	if s.shifts != nil && (in.Method == "efectivo" || in.Method == "tarjeta_debito" || in.Method == "tarjeta_credito") {
		shiftID, _ = s.shifts.OpenShiftIDForCashier(createdBy)
	}
	return s.store.AddPayment(accountID, in, createdBy, shiftID)
}

func (s *Service) CloseAccount(id, closedBy int) (Account, error) {
	account, err := s.store.GetAccount(id)
	if err != nil {
		return Account{}, err
	}
	if account.Status != "abierta" {
		return Account{}, fmt.Errorf("%w: la cuenta ya está cerrada", ErrValidation)
	}
	if account.Balance > balanceTolerance {
		return Account{}, fmt.Errorf("%w: la cuenta tiene un saldo pendiente de %.2f", ErrValidation, account.Balance)
	}
	closed, err := s.store.CloseAccount(id, closedBy)
	if err != nil {
		return Account{}, err
	}
	// Libera la mesa solo si ya no queda nada por cobrar en ella: ninguna
	// otra cuenta abierta (cuentas divididas) ni partidas sin cuenta.
	if s.releaser != nil && closed.TableID != nil {
		pending, err := s.store.TableHasPendingBilling(*closed.TableID)
		if err != nil {
			return Account{}, err
		}
		if !pending {
			if err := s.releaser.Release(*closed.TableID); err != nil {
				return Account{}, err
			}
		}
	}
	return closed, nil
}

func (s *Service) GetTicket(id int) (Ticket, error) {
	account, err := s.store.GetAccount(id)
	if err != nil {
		return Ticket{}, err
	}
	return Ticket{Account: account, Note: "Este ticket no tiene validez fiscal salvo que se emita CFDI."}, nil
}
