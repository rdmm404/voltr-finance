package webui

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

//go:embed assets/dist/*
var assetFiles embed.FS

type Handler struct {
	config    Config
	dashboard *Dashboard
	logger    *slog.Logger
	now       func() time.Time
}

func New(config Config, services Services, logger *slog.Logger) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if services.Budgets == nil || services.Users == nil || services.Households == nil {
		return nil, fmt.Errorf("dashboard services are required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{config: config, dashboard: NewDashboard(services), logger: logger, now: time.Now}, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	assets, _ := fs.Sub(assetFiles, "assets/dist")
	assetHandler := http.StripPrefix("/assets/", http.FileServerFS(assets))
	mux.Handle("GET /assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		assetHandler.ServeHTTP(w, r)
	}))
	mux.HandleFunc("GET /", h.dashboardPage)
}

func (h *Handler) dashboardPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.renderStatus(w, http.StatusNotFound, "Page not found", "The requested page does not exist.")
		return
	}
	state, redirect, err := ParseRequestState(r.URL.Query(), h.config, h.now())
	if err != nil {
		h.renderStatus(w, http.StatusBadRequest, "Invalid dashboard request", err.Error())
		return
	}
	if redirect {
		http.Redirect(w, r, StateURL(state), http.StatusSeeOther)
		return
	}
	view, err := h.dashboard.Assemble(r.Context(), state)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			h.renderStatus(w, http.StatusNotFound, "Owner not found", "The selected user or household does not exist.")
			return
		}
		h.logger.ErrorContext(r.Context(), "render finance dashboard", "error", err)
		h.renderStatus(w, http.StatusInternalServerError, "Dashboard unavailable", "The dashboard could not be loaded safely. Please try again.")
		return
	}
	h.render(r.Context(), w, http.StatusOK, DashboardPage(view))
}

func (h *Handler) renderStatus(w http.ResponseWriter, status int, title, message string) {
	h.render(context.Background(), w, status, StatusPage(status, title, message))
}
func (h *Handler) render(ctx context.Context, w http.ResponseWriter, status int, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := component.Render(ctx, w); err != nil {
		h.logger.Error("render HTML", "error", err)
	}
}

func stateClass(state SemanticState) string {
	switch state {
	case StateDanger:
		return "border-danger text-danger"
	case StateWarning:
		return "border-warning text-warning"
	default:
		return "border-positive text-positive"
	}
}
func selected(value, selected int64) bool { return value == selected }
func join(parts ...string) string         { return strings.Join(parts, " ") }
