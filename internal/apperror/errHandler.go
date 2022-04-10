package apperror

import (
	"errors"
	"net/http"
)

func HandleError(w http.ResponseWriter, err error) {
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			if errors.Is(err, ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write(ErrNotFound.Marshal())
				return
			} else if errors.Is(err, ErrAlreadyExist) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write(ErrNotFound.Marshal())
				return
			}
			err := err.(*AppError)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(err.Marshal())
			return
		}
		w.WriteHeader(418)
		_, _ = w.Write(systemError(err.Error()).Marshal())
	}
}
