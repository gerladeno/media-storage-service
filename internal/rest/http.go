package rest

import (
	"compress/flate"
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gerladeno/media-storage-service/internal/storage"
	"github.com/gerladeno/media-storage-service/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sirupsen/logrus"
)

type Service interface {
	GetFile(ctx context.Context, noteUUID, fileName string) (f *storage.File, err error)
	GetFilesByNoteUUID(ctx context.Context, noteUUID string) ([]*storage.File, error)
	Create(ctx context.Context, noteUUID string, dto storage.CreateFileDTO) error
	Delete(ctx context.Context, noteUUID, fileName string) error
}

const gitURL = "https://github.com/gerladeno/media-storage-service"

func NewRouter(log *logrus.Logger, service Service, key *rsa.PublicKey, host, version string) chi.Router {
	handler := newHandler(log, service, key)
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cors.AllowAll().Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.NewCompressor(flate.DefaultCompression).Handler)
	r.NotFound(notFoundHandler)
	r.Get("/ping", pingHandler)
	r.Get("/version", versionHandler(version))
	r.Group(func(r chi.Router) {
		r.Use(metrics.NewPromMiddleware(host))
		r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log, NoColor: true}))
		r.Use(middleware.Timeout(30 * time.Second))
		r.Use(middleware.Throttle(100))
		r.Use(handler.auth)
		r.Route("/public", func(r chi.Router) {
			r.Route("/v1", func(r chi.Router) {
				r.Get("/api/files/{id}", handler.getFile)
				r.Get("/api/files", handler.getFilesByNoteUUID)
				r.Post("/api/files", handler.createFile)
				r.Delete("/api/files/{id}", handler.deleteFile)
				r.Group(func(r chi.Router) {
					// protected endpoints
				})
			})
		})
		r.Route("/private", func(r chi.Router) {
		})
	})
	return r
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "404 page not found. Check docs: "+gitURL, http.StatusNotFound)
}

func pingHandler(w http.ResponseWriter, _ *http.Request) {
	writeResponse(w, "pong")
}

func versionHandler(version string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeResponse(w, version)
	}
}

func writeResponse(w http.ResponseWriter, data interface{}) {
	response := JSONResponse{Data: data}
	_ = json.NewEncoder(w).Encode(response) //nolint:errchkjson
}

func writeErrResponse(w http.ResponseWriter, message string, status int) {
	response := JSONResponse{Data: []int{}, Error: &message, Code: &status}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response) //nolint:errchkjson
}

type JSONResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Meta  *Meta       `json:"meta,omitempty"`
	Error *string     `json:"error,omitempty"`
	Code  *int        `json:"code,omitempty"`
}

type Meta struct {
	Count int `json:"count"`
}
