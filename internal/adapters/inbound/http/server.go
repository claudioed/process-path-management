package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/riandyrn/otelchi"
	otelchimetric "github.com/riandyrn/otelchi/metric"

	"github.com/claudioed/process-path-management/internal/application/usecases"
	"github.com/claudioed/process-path-management/internal/domain/processpath"
	"github.com/claudioed/process-path-management/internal/domain/shared"
)

// DefaultServiceName labels this service in logs, spans and metrics when
// the caller does not supply one. It matches the OTel resource's
// service.name.
const DefaultServiceName = "process-path-management"

// Server holds every use case the HTTP adapter depends on.
type Server struct {
	DefinePath     *usecases.DefinePath
	RevisePath     *usecases.RevisePath
	DeactivatePath *usecases.DeactivatePath
	GetPath        *usecases.GetPath
	ListPaths      *usecases.ListPaths
}

// NewRouter builds the chi router for this service's REST API. A nil
// logger defaults to slog.Default(); an empty serviceName defaults to
// DefaultServiceName.
//
// Middleware order matters here (fleet-standard-metrics ADR, Tier 1 item
// 2): otelchi runs before RequestLogger so the request context already
// carries a span by the time a line is logged. WithChiRoutes resolves
// the route pattern up front, so spans/metrics are labeled
// "/process-paths/{pathId}" rather than one distinct name per path id.
func NewRouter(s *Server, logger *slog.Logger, serviceName string) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if serviceName == "" {
		serviceName = DefaultServiceName
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(otelchi.Middleware(serviceName, otelchi.WithChiRoutes(r)))
	// Emits http.server.request.duration (seconds) per OTel HTTP semantic
	// conventions; no hand-rolled histogram needed.
	r.Use(otelchimetric.NewServerRequestDuration(otelchimetric.NewBaseConfig(serviceName)))
	r.Use(RequestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware())

	r.Get("/healthz", s.handleHealthz)

	r.Post("/process-paths", s.handleDefinePath)
	r.Get("/process-paths", s.handleListPaths)
	r.Get("/process-paths/{pathId}", s.handleGetPath)
	r.Put("/process-paths/{pathId}", s.handleRevisePath)
	r.Delete("/process-paths/{pathId}", s.handleDeactivatePath)

	return r
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDefinePath(w http.ResponseWriter, r *http.Request) {
	var req defineProcessPathRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	p, err := s.DefinePath.Execute(r.Context(), shared.PathId(req.PathId), req.MatchPrefix, req.Direct, toCapabilities(req.RequiredCapabilities))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProcessPathResponse(p))
}

func (s *Server) handleListPaths(w http.ResponseWriter, r *http.Request) {
	// activeOnly is the default (?all=true opts into the audit view) —
	// matches the retired YAML catalogue's own posture that every
	// consumer's normal read is "the currently valid set", not
	// everything that ever existed.
	activeOnly := r.URL.Query().Get("all") != "true"

	paths, err := s.ListPaths.Execute(r.Context(), activeOnly)
	if err != nil {
		writeError(w, r, err)
		return
	}
	responses := make([]processPathResponse, len(paths))
	for i, p := range paths {
		responses[i] = toProcessPathResponse(p)
	}
	writeJSON(w, http.StatusOK, responses)
}

func (s *Server) handleGetPath(w http.ResponseWriter, r *http.Request) {
	pathId := shared.PathId(chi.URLParam(r, "pathId"))

	p, err := s.GetPath.Execute(r.Context(), pathId)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProcessPathResponse(p))
}

func (s *Server) handleRevisePath(w http.ResponseWriter, r *http.Request) {
	pathId := shared.PathId(chi.URLParam(r, "pathId"))

	var req reviseProcessPathRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	p, err := s.RevisePath.Execute(r.Context(), pathId, req.MatchPrefix, toCapabilities(req.RequiredCapabilities))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProcessPathResponse(p))
}

func (s *Server) handleDeactivatePath(w http.ResponseWriter, r *http.Request) {
	pathId := shared.PathId(chi.URLParam(r, "pathId"))

	if err := s.DeactivatePath.Execute(r.Context(), pathId); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const timeFormat = time.RFC3339

func toProcessPathResponse(p *processpath.ProcessPath) processPathResponse {
	caps := p.RequiredCapabilities()
	strCaps := make([]string, len(caps))
	for i, c := range caps {
		strCaps[i] = string(c)
	}
	return processPathResponse{
		PathId:               string(p.ID()),
		MatchPrefix:          p.MatchPrefix(),
		Direct:               p.Direct(),
		RequiredCapabilities: strCaps,
		Status:               string(p.Status()),
		CreatedAt:            p.CreatedAt().UTC().Format(timeFormat),
		UpdatedAt:            p.UpdatedAt().UTC().Format(timeFormat),
	}
}

func toCapabilities(ss []string) []shared.Capability {
	out := make([]shared.Capability, len(ss))
	for i, s := range ss {
		out[i] = shared.Capability(s)
	}
	return out
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeProblem(w, http.StatusBadRequest, problemInfo{"malformed-request-body", "The request body is not valid JSON"}, err.Error(), r.URL.Path)
		return false
	}
	return true
}

// writeError writes a domain/application error as an RFC 7807
// (application/problem+json) response.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	writeProblem(w, statusFor(err), problemFor(err), err.Error(), r.URL.Path)
}

func writeProblem(w http.ResponseWriter, status int, info problemInfo, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problemDetails{
		Type:     problemBaseURI + info.slug,
		Title:    info.title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// corsMiddleware allows the operator-facing SPA (this service's own
// future path-mgmt-mfe remote, plus warehouse-console) to call this API
// directly from the browser. CORS_ALLOWED_ORIGINS overrides the
// local-dev default (comma-separated) for staging/prod deployments.
// Includes PUT/DELETE (unlike a read-mostly service's CORS policy)
// since operators mutate paths directly from the browser.
func corsMiddleware() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins(),
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}

// allowedOrigins resolves the browser origins permitted to call this
// service, from CORS_ALLOWED_ORIGINS (comma-separated) or the local-dev
// default.
func allowedOrigins() []string {
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{"http://localhost:5173", "http://localhost:5189"}
}
