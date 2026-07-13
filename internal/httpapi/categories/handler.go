package categories

import (
	"context"
	"net/http"

	"rdmm404/voltr-finance/internal/api"
	appcategories "rdmm404/voltr-finance/internal/app/categories"
	"rdmm404/voltr-finance/internal/httpapi"
)

type service interface {
	Create(context.Context, appcategories.CreateInput) (appcategories.Category, error)
	List(context.Context, bool) ([]appcategories.Category, error)
	GetByCode(context.Context, string) (appcategories.Category, error)
	Update(context.Context, appcategories.UpdateInput) (appcategories.Category, error)
	Deactivate(context.Context, string) (appcategories.Category, error)
}

type Handler struct{ service service }

func New(service service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(router *httpapi.Router) {
	router.HandleFunc(http.MethodPost, api.CategoriesPath, h.create)
	router.HandleFunc(http.MethodGet, api.CategoriesPath, h.list)
	router.HandleFunc(http.MethodGet, api.CategoryPath, h.get)
	router.HandleFunc(http.MethodDelete, api.CategoryPath, h.deactivate)
	router.HandleFunc(http.MethodPatch, api.CategoryUpdatePath, h.update)
}

func (h *Handler) create(w http.ResponseWriter, request *http.Request) {
	var body api.CreateCategoryRequest
	if !decode(w, request, &body) {
		return
	}
	item, err := h.service.Create(request.Context(), appcategories.CreateInput{
		Name: body.Name, Code: body.Code, Description: body.Description,
	})
	if err != nil {
		fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, category(item))
}

func (h *Handler) list(w http.ResponseWriter, request *http.Request) {
	includeInactive, err := httpapi.QueryBool(request, "includeInactive", false)
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	items, err := h.service.List(request.Context(), includeInactive)
	if err != nil {
		fail(w, request, err)
		return
	}
	response := make([]api.Category, 0, len(items))
	for _, item := range items {
		response = append(response, category(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) get(w http.ResponseWriter, request *http.Request) {
	item, err := h.service.GetByCode(request.Context(), request.PathValue("code"))
	if err != nil {
		fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, category(item))
}

func (h *Handler) update(w http.ResponseWriter, request *http.Request) {
	id, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	var body api.UpdateCategoryRequest
	if !decode(w, request, &body) {
		return
	}
	item, err := h.service.Update(request.Context(), appcategories.UpdateInput{
		ID: id, Name: body.Name, Description: body.Description, ClearDescription: body.ClearDescription,
	})
	if err != nil {
		fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, category(item))
}

func (h *Handler) deactivate(w http.ResponseWriter, request *http.Request) {
	item, err := h.service.Deactivate(request.Context(), request.PathValue("code"))
	if err != nil {
		fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, category(item))
}

func category(item appcategories.Category) api.Category {
	return api.Category{ID: item.ID, Code: item.Code, Name: item.Name, Description: item.Description, IsActive: item.IsActive}
}

func decode(w http.ResponseWriter, request *http.Request, value any) bool {
	if err := httpapi.DecodeJSON(w, request, value); err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return false
	}
	return true
}

func fail(w http.ResponseWriter, request *http.Request, err error) {
	httpapi.WriteApplicationError(w, request, nil, err)
}
