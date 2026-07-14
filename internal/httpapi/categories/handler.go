package categories

import (
	"context"
	"net/http"

	"rdmm404/voltr-finance/internal/api"
	appcategories "rdmm404/voltr-finance/internal/app/categories"
	"rdmm404/voltr-finance/internal/httpapi"
)

type Service interface {
	Create(context.Context, appcategories.CreateInput) (appcategories.Category, error)
	List(context.Context, bool) ([]appcategories.Category, error)
	GetByCode(context.Context, string) (appcategories.Category, error)
	Update(context.Context, appcategories.UpdateInput) (appcategories.Category, error)
	Deactivate(context.Context, string) (appcategories.Category, error)
}

type Handler struct {
	service Service
	support *httpapi.HandlerSupport
}

func New(service Service, support ...*httpapi.HandlerSupport) *Handler {
	return &Handler{service: service, support: httpapi.HandlerSupportOrDefault(support...)}
}

func (h *Handler) Register(router *httpapi.Router) {
	router.HandleFunc(http.MethodPost, api.CategoriesPath, h.create)
	router.HandleFunc(http.MethodGet, api.CategoriesPath, h.list)
	router.HandleFunc(http.MethodGet, api.CategoryPath, h.get)
	router.HandleFunc(http.MethodDelete, api.CategoryPath, h.deactivate)
	router.HandleFunc(http.MethodPatch, api.CategoryPath, h.update)
}

func (h *Handler) create(w http.ResponseWriter, request *http.Request) {
	var body api.CreateCategoryRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	item, err := h.service.Create(request.Context(), appcategories.CreateInput{
		Name: body.Name, Code: body.Code, Description: body.Description,
	})
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, category(item))
}

func (h *Handler) list(w http.ResponseWriter, request *http.Request) {
	query, err := listQuery(request)
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	items, err := h.service.List(request.Context(), query.IncludeInactive)
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	response := make([]api.Category, 0, len(items))
	for _, item := range items {
		response = append(response, category(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

func listQuery(request *http.Request) (api.ListCategoriesQuery, error) {
	includeInactive, err := httpapi.QueryBool(request, "includeInactive", false)
	return api.ListCategoriesQuery{IncludeInactive: includeInactive}, err
}

func (h *Handler) get(w http.ResponseWriter, request *http.Request) {
	item, err := h.service.GetByCode(request.Context(), request.PathValue("code"))
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, category(item))
}

func (h *Handler) update(w http.ResponseWriter, request *http.Request) {
	var body api.UpdateCategoryRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	description, err := httpapi.NullablePatch(body.Description, body.ClearDescription, "description")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	item, err := h.service.Update(request.Context(), appcategories.UpdateInput{Code: request.PathValue("code"), Name: body.Name, Description: description})
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, category(item))
}

func (h *Handler) deactivate(w http.ResponseWriter, request *http.Request) {
	item, err := h.service.Deactivate(request.Context(), request.PathValue("code"))
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, category(item))
}

func category(item appcategories.Category) api.Category {
	return api.Category{ID: item.ID, Code: item.Code, Name: item.Name, Description: item.Description, IsActive: item.IsActive}
}
