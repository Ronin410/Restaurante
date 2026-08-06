package orders

import (
	"errors"
	"fmt"
)

var ErrValidation = errors.New("datos inválidos")

var validTypes = map[string]bool{"mesa": true, "llevar": true, "domicilio": true}
var validKitchenStatuses = map[string]bool{"preparando": true, "listo": true, "entregado": true}

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateOrder(waiterID int, in CreateOrderInput) (Order, error) {
	if !validTypes[in.Type] {
		return Order{}, fmt.Errorf("%w: tipo de pedido inválido (mesa, llevar o domicilio)", ErrValidation)
	}
	if in.Type == "mesa" && in.TableID == nil {
		return Order{}, fmt.Errorf("%w: se requiere la mesa para pedidos en mesa", ErrValidation)
	}
	if in.Type == "domicilio" && (in.CustomerAddress == "" || in.CustomerPhone == "") {
		return Order{}, fmt.Errorf("%w: domicilio y teléfono son obligatorios para pedidos a domicilio", ErrValidation)
	}
	for _, item := range in.Items {
		if item.ProductID == 0 {
			return Order{}, fmt.Errorf("%w: producto obligatorio en cada partida", ErrValidation)
		}
	}

	orderID, err := s.store.CreateOrder(in.TableID, in.Type, waiterID, in.CustomerName, in.CustomerPhone, in.CustomerAddress)
	if err != nil {
		return Order{}, err
	}
	for _, item := range in.Items {
		if _, err := s.store.AddItem(orderID, 1, item); err != nil {
			return Order{}, err
		}
	}
	return s.store.GetOrder(orderID)
}

// AddItems agrega productos a una comanda ya abierta sin duplicar ticket:
// se registran como una nueva "tanda" para que cocina identifique lo que se
// agregó después del envío original.
func (s *Service) AddItems(orderID int, in AddItemsInput) (Order, error) {
	order, err := s.store.GetOrder(orderID)
	if err != nil {
		return Order{}, err
	}
	if order.Status != "abierta" {
		return Order{}, fmt.Errorf("%w: la comanda ya está cerrada", ErrValidation)
	}
	if len(in.Items) == 0 {
		return Order{}, fmt.Errorf("%w: se requiere al menos un producto", ErrValidation)
	}
	for _, item := range in.Items {
		if item.ProductID == 0 {
			return Order{}, fmt.Errorf("%w: producto obligatorio en cada partida", ErrValidation)
		}
	}
	maxTanda, err := s.store.MaxTanda(orderID)
	if err != nil {
		return Order{}, err
	}
	nextTanda := maxTanda + 1
	for _, item := range in.Items {
		if _, err := s.store.AddItem(orderID, nextTanda, item); err != nil {
			return Order{}, err
		}
	}
	return s.store.GetOrder(orderID)
}

func (s *Service) GetOrder(id int) (Order, error) {
	return s.store.GetOrder(id)
}

func (s *Service) ListOpenOrders(tableID *int) ([]Order, error) {
	return s.store.ListOpenOrders(tableID)
}

func (s *Service) ListKitchenItems() ([]OrderItem, error) {
	return s.store.ListKitchenItems()
}

// UpdateItemStatus mueve una partida por el flujo de cocina: preparando ->
// listo -> entregado.
func (s *Service) UpdateItemStatus(itemID int, status string) (OrderItem, error) {
	if !validKitchenStatuses[status] {
		return OrderItem{}, fmt.Errorf("%w: estatus inválido", ErrValidation)
	}
	item, err := s.store.GetItem(itemID)
	if err != nil {
		return OrderItem{}, err
	}
	if item.Status == "cancelado" || item.Status == "entregado" {
		return OrderItem{}, fmt.Errorf("%w: la partida ya no puede cambiar de estatus", ErrValidation)
	}
	return s.store.UpdateItemStatus(itemID, status)
}

// CancelItem cancela o modifica una partida. El endpoint solo debe estar
// disponible para roles supervisores (gerente/admin) para dejar control de
// mermas/fraude; aquí se valida el motivo obligatorio.
func (s *Service) CancelItem(itemID int, reason string, authorizedBy int) (OrderItem, error) {
	if reason == "" {
		return OrderItem{}, fmt.Errorf("%w: el motivo de cancelación es obligatorio", ErrValidation)
	}
	item, err := s.store.GetItem(itemID)
	if err != nil {
		return OrderItem{}, err
	}
	if item.Status == "cancelado" {
		return OrderItem{}, fmt.Errorf("%w: la partida ya estaba cancelada", ErrValidation)
	}
	if item.AccountID != nil {
		return OrderItem{}, fmt.Errorf("%w: la partida ya forma parte de una cuenta cerrada/generada", ErrValidation)
	}
	return s.store.CancelItem(itemID, reason, authorizedBy)
}

func (s *Service) CloseOrder(id int) error {
	return s.store.CloseOrder(id)
}
