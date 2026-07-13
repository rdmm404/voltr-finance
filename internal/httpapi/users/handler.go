package users

import (
	"context"
	"net/http"

	"rdmm404/voltr-finance/internal/api"
	appusers "rdmm404/voltr-finance/internal/app/users"
	"rdmm404/voltr-finance/internal/httpapi"
)

type service interface {
	Create(context.Context, appusers.CreateInput) (appusers.User, error)
	Update(context.Context, appusers.UpdateInput) (appusers.User, error)
	Get(context.Context, int64) (appusers.User, error)
	Resolve(context.Context, appusers.Selector) (appusers.User, error)
	List(context.Context) ([]appusers.User, error)
}

type Handler struct {
	service service
	support *httpapi.HandlerSupport
}

func New(service service, support ...*httpapi.HandlerSupport) *Handler {
	return &Handler{service: service, support: httpapi.HandlerSupportOrDefault(support...)}
}
func (h *Handler) Register(router *httpapi.Router) {
	router.HandleFunc(http.MethodPost, api.UsersPath, h.create)
	router.HandleFunc(http.MethodGet, api.UsersPath, h.list)
	router.HandleFunc(http.MethodPost, api.UserResolvePath, h.resolve)
	router.HandleFunc(http.MethodGet, api.UserPath, h.get)
	router.HandleFunc(http.MethodPatch, api.UserPath, h.update)
}
func (h *Handler) create(w http.ResponseWriter, request *http.Request) {
	var body api.CreateUserRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	item, err := h.service.Create(request.Context(), appusers.CreateInput{Name: body.Name, DiscordID: body.DiscordID, TelegramID: body.TelegramID, PhoneNumber: body.PhoneNumber, WhatsAppID: body.WhatsAppID})
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, user(item))
}
func (h *Handler) list(w http.ResponseWriter, request *http.Request) {
	items, err := h.service.List(request.Context())
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
	httpapi.WriteJSON(w, http.StatusOK, user(item))
}
func (h *Handler) resolve(w http.ResponseWriter, request *http.Request) {
	var body api.ResolveUserRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	item, err := h.service.Resolve(request.Context(), selector(body.IdentitySelector))
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, user(item))
}
func (h *Handler) update(w http.ResponseWriter, request *http.Request) {
	id, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	var body api.UpdateUserRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	input, err := updateInput(id, body)
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	item, err := h.service.Update(request.Context(), input)
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, user(item))
}
func updateInput(id int64, body api.UpdateUserRequest) (appusers.UpdateInput, error) {
	discordID, err := httpapi.NullablePatch(body.DiscordID, body.ClearDiscordID, "discordId")
	if err != nil {
		return appusers.UpdateInput{}, err
	}
	telegramID, err := httpapi.NullablePatch(body.TelegramID, body.ClearTelegramID, "telegramId")
	if err != nil {
		return appusers.UpdateInput{}, err
	}
	phoneNumber, err := httpapi.NullablePatch(body.PhoneNumber, body.ClearPhoneNumber, "phoneNumber")
	if err != nil {
		return appusers.UpdateInput{}, err
	}
	whatsAppID, err := httpapi.NullablePatch(body.WhatsAppID, body.ClearWhatsAppID, "whatsappId")
	if err != nil {
		return appusers.UpdateInput{}, err
	}
	return appusers.UpdateInput{ID: id, Name: body.Name, DiscordID: discordID, TelegramID: telegramID, PhoneNumber: phoneNumber, WhatsAppID: whatsAppID}, nil
}

func selector(value api.IdentitySelector) appusers.Selector {
	return appusers.Selector{UserID: value.UserID, DiscordID: value.DiscordID, TelegramID: value.TelegramID, PhoneNumber: value.PhoneNumber, WhatsAppID: value.WhatsAppID}
}
func user(item appusers.User) api.User {
	return api.User{ID: item.ID, Name: item.Name, DiscordID: item.DiscordID, TelegramID: item.TelegramID, PhoneNumber: item.PhoneNumber, WhatsAppID: item.WhatsAppID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
