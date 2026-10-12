package catalog

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ronin410/restaurante/backend/internal/audit"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/httpx"
)

type Handler struct {
	svc   *Service
	audit *audit.Logger
}

func NewHandler(svc *Service, audit *audit.Logger) *Handler {
	return &Handler{svc: svc, audit: audit}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	manage := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente)

	r.Route("/categories", func(r chi.Router) {
		r.Get("/", h.listCategories)
		r.With(manage).Post("/", h.createCategory)
		r.With(manage).Put("/{id}", h.updateCategory)
		r.With(manage).Delete("/{id}", h.deleteCategory)
	})

	r.Route("/modifier-groups", func(r chi.Router) {
		r.Get("/", h.listModifierGroups)
		r.With(manage).Post("/", h.createModifierGroup)
		r.With(manage).Put("/{id}", h.updateModifierGroup)
		r.With(manage).Delete("/{id}", h.deleteModifierGroup)
		r.With(manage).Post("/{id}/options", h.addModifierOption)
		r.With(manage).Put("/{id}/options/{optionId}", h.updateModifierOption)
		r.With(manage).Delete("/{id}/options/{optionId}", h.deleteModifierOption)
	})

	r.Route("/products", func(r chi.Router) {
		r.Get("/", h.listProducts)
		r.Get("/{id}", h.getProduct)
		r.With(manage).Post("/", h.createProduct)
		r.With(manage).Put("/{id}", h.updateProduct)
		r.With(manage).Delete("/{id}", h.deleteProduct)
	})
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListCategories()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var in CategoryInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	c, err := h.svc.CreateCategory(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in CategoryInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	c, err := h.svc.UpdateCategory(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.svc.DeleteCategory(id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listModifierGroups(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListModifierGroups()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) createModifierGroup(w http.ResponseWriter, r *http.Request) {
	var in ModifierGroupInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	g, err := h.svc.CreateModifierGroup(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, g)
}

func (h *Handler) addModifierOption(w http.ResponseWriter, r *http.Request) {
	groupID, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in ModifierOptionInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	o, err := h.svc.AddModifierOption(groupID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, o)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListProducts(r.URL.Query().Get("active") == "1")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	p, err := h.svc.GetProduct(id)
	if err != nil {
		if errors.Is(err, ErrInUse) {
			httpx.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var in ProductInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.svc.CreateProduct(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "create", "product", p.ID, p.Name)
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in ProductInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.svc.UpdateProduct(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "update", "product", id, p.Name)
	httpx.JSON(w, http.StatusOK, p)
}

func idParam(r *http.Request) (int, error) {
	return strconv.Atoi(chi.URLParam(r, "id"))
}

func writeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrValidation) {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, ErrInUse) {
		httpx.Error(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	httpx.Error(w, http.StatusInternalServerError, err.Error())
}

func (h *Handler) updateModifierGroup(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in ModifierGroupInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	g, err := h.svc.UpdateModifierGroup(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}

func (h *Handler) deleteModifierGroup(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.svc.DeleteModifierGroup(id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func optionParams(r *http.Request) (int, int, error) {
	g, err := idParam(r)
	if err != nil {
		return 0, 0, err
	}
	o, err := strconv.Atoi(chi.URLParam(r, "optionId"))
	return g, o, err
}

func (h *Handler) updateModifierOption(w http.ResponseWriter, r *http.Request) {
	g, o, err := optionParams(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in ModifierOptionInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	opt, err := h.svc.UpdateModifierOption(g, o, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, opt)
}

func (h *Handler) deleteModifierOption(w http.ResponseWriter, r *http.Request) {
	g, o, err := optionParams(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.svc.DeleteModifierOption(g, o); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.svc.DeleteProduct(id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
