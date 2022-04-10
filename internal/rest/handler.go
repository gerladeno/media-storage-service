package rest

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gerladeno/media-storage-service/internal/apperror"
	"github.com/gerladeno/media-storage-service/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type handler struct {
	log     *logrus.Entry
	service Service
	key     *rsa.PublicKey
}

func newHandler(log *logrus.Logger, service Service, key *rsa.PublicKey) *handler {
	return &handler{
		log:     log.WithField("module", "rest"),
		service: service,
		key:     key,
	}
}

func (h *handler) getFile(w http.ResponseWriter, r *http.Request) {
	noteUUID := r.URL.Query().Get("note_uuid")
	if noteUUID == "" {
		apperror.HandleError(w, apperror.BadRequestError("note_uuid query parameter is required"))
		return
	}
	fileID := chi.URLParam(r, "id")
	f, err := h.service.GetFile(r.Context(), noteUUID, fileID)
	if err != nil {
		apperror.HandleError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", f.Name))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))

	_, _ = w.Write(f.Bytes)
}

func (h *handler) getFilesByNoteUUID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "form/json")
	noteUUID := r.URL.Query().Get("note_uuid")
	if noteUUID == "" {
		apperror.HandleError(w, apperror.BadRequestError("note_uuid query parameter is required"))
		return
	}
	file, err := h.service.GetFilesByNoteUUID(r.Context(), noteUUID)
	if err != nil {
		apperror.HandleError(w, err)
		return
	}
	filesBytes, err := json.Marshal(file)
	if err != nil {
		apperror.HandleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(filesBytes)
}

func (h *handler) createFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "form/json")
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		apperror.HandleError(w, err)
		return
	}
	files, ok := r.MultipartForm.File["file"]
	if !ok || len(files) == 0 {
		apperror.HandleError(w, apperror.BadRequestError("file required"))
		return
	}
	fileInfo := files[0]
	fileReader, err := fileInfo.Open()
	if err != nil {
		apperror.HandleError(w, err)
		return
	}
	dto := storage.CreateFileDTO{
		Name:   fileInfo.Filename,
		Size:   fileInfo.Size,
		Reader: fileReader,
	}
	err = h.service.Create(r.Context(), r.Form.Get("note_uuid"), dto)
	if err != nil {
		apperror.HandleError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fileID := chi.URLParam(r, "id")
	noteUUID := r.URL.Query().Get("note_uuid")
	if noteUUID == "" {
		apperror.HandleError(w, apperror.BadRequestError("note_uuid query parameter is required"))
		return
	}

	err := h.service.Delete(r.Context(), noteUUID, fileID)
	if err != nil {
		apperror.HandleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
