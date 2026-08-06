package catalog

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

func (s *Service) ListCategories() ([]Category, error) { return s.store.ListCategories() }

func (s *Service) CreateCategory(in CategoryInput) (Category, error) {
	if in.Name == "" {
		return Category{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	return s.store.CreateCategory(in)
}

func (s *Service) UpdateCategory(id int, in CategoryInput) (Category, error) {
	if in.Name == "" {
		return Category{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	return s.store.UpdateCategory(id, in)
}

func (s *Service) DeleteCategory(id int) error { return s.store.DeleteCategory(id) }

func (s *Service) ListModifierGroups() ([]ModifierGroup, error) { return s.store.ListModifierGroups() }

func (s *Service) CreateModifierGroup(in ModifierGroupInput) (ModifierGroup, error) {
	if in.Name == "" {
		return ModifierGroup{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	return s.store.CreateModifierGroup(in)
}

func (s *Service) AddModifierOption(groupID int, in ModifierOptionInput) (ModifierOption, error) {
	if in.Name == "" {
		return ModifierOption{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	return s.store.AddModifierOption(groupID, in)
}

func (s *Service) ListProducts(activeOnly bool) ([]Product, error) {
	return s.store.ListProducts(activeOnly)
}

func (s *Service) GetProduct(id int) (Product, error) { return s.store.GetProduct(id) }

func (s *Service) CreateProduct(in ProductInput) (Product, error) {
	if in.Name == "" || in.CategoryID == 0 {
		return Product{}, fmt.Errorf("%w: nombre y categoría obligatorios", ErrValidation)
	}
	if in.Price < 0 {
		return Product{}, fmt.Errorf("%w: el precio no puede ser negativo", ErrValidation)
	}
	return s.store.CreateProduct(in)
}

func (s *Service) UpdateProduct(id int, in ProductInput) (Product, error) {
	if in.Name == "" || in.CategoryID == 0 {
		return Product{}, fmt.Errorf("%w: nombre y categoría obligatorios", ErrValidation)
	}
	if in.Price < 0 {
		return Product{}, fmt.Errorf("%w: el precio no puede ser negativo", ErrValidation)
	}
	return s.store.UpdateProduct(id, in)
}
