package households

import (
	"context"
	"net/http"

	"rdmm404/voltr-finance/internal/api"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	"rdmm404/voltr-finance/internal/httpapi"
)

type Service interface {
	List(context.Context) ([]apphouseholds.Household, error)
	Get(context.Context, int64) (apphouseholds.Household, error)
	Resolve(context.Context, apphouseholds.Selector) (apphouseholds.Household, error)
	ListUsers(context.Context, int64) ([]apphouseholds.User, error)
}
type Handler struct {
	service Service
	support *httpapi.HandlerSupport
}

func New(service Service, support ...*httpapi.HandlerSupport) *Handler {
	return &Handler{service: service, support: httpapi.HandlerSupportOrDefault(support...)}
}
func (h *Handler) Register(router *httpapi.Router) {
	router.HandleFunc(http.MethodGet, api.HouseholdsPath, h.list)
	router.HandleFunc(http.MethodGet, api.HouseholdResolvePath, h.resolve)
	router.HandleFunc(http.MethodGet, api.HouseholdPath, h.get)
	router.HandleFunc(http.MethodGet, api.HouseholdUsersPath, h.listUsers)
}
func (h *Handler) list(w http.ResponseWriter, request *http.Request) {
	items, err := h.service.List(request.Context())
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	response := make([]api.Household, 0, len(items))
	for _, item := range items {
		response = append(response, household(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}
func (h *Handler) get(w http.ResponseWriter, request *http.Request) {
	id, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	item, err := h.service.Get(request.Context(), id)
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, household(item))
}
func (h *Handler) resolve(w http.ResponseWriter, request *http.Request) {
	item, err := h.service.Resolve(request.Context(), apphouseholds.Selector{Name: httpapi.QueryString(request, "name"), GuildID: httpapi.QueryString(request, "guildId")})
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, household(item))
}
func (h *Handler) listUsers(w http.ResponseWriter, request *http.Request) {
	id, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	items, err := h.service.ListUsers(request.Context(), id)
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	response := make([]api.User, 0, len(items))
	for _, item := range items {
		response = append(response, user(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}
func household(item apphouseholds.Household) api.Household {
	return api.Household{ID: item.ID, Name: item.Name, GuildID: item.GuildID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func user(item apphouseholds.User) api.User {
	return api.User{ID: item.ID, Name: item.Name, DiscordID: item.DiscordID, TelegramID: item.TelegramID, PhoneNumber: item.PhoneNumber, WhatsAppID: item.WhatsAppID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
