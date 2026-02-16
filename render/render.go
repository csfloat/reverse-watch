package render

import (
	stderrors "errors"
	"net/http"
	"strings"

	"reverse-watch/errors"
	"reverse-watch/logging"

	"github.com/go-chi/render"
	"gorm.io/gorm"
)

func isUniqueConstrainError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	var e *errors.Error
	if !stderrors.As(err, &e) {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			e = errors.New(errors.NotFound, err.Error())
		} else if isUniqueConstrainError(err) {
			e = errors.New(errors.Conflict, err.Error())
		} else {
			logging.Log.Warnf("attempting to render non-server error: %v", err)
			e = errors.New(errors.InternalServerError, err.Error())
		}
	}

	if err := render.Render(w, r, e); err != nil {
		logging.Log.Errorf("failed to render error: %v", err)
	}
}

func Errorf(w http.ResponseWriter, r *http.Request, err errors.Error, f string, args ...interface{}) {
	Error(w, r, errors.Newf(err, nil, f, args...))
}

func JSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	render.JSON(w, r, v)
}

func Success(w http.ResponseWriter, r *http.Request, message string) {
	JSON(w, r, message)
}
