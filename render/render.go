package render

import (
	stderrors "errors"
	"net/http"

	"reverse-watch/errors"
	"reverse-watch/logging"

	"github.com/go-chi/render"
)

var log = logging.Log

func Error(w http.ResponseWriter, r *http.Request, err error) {
	var e *errors.Error
	if !stderrors.As(err, &e) {
		log.Warnf("attempting to render non-server error: %v", err)
		e = errors.New(errors.Unknown, err.Error())
	}

	if err := render.Render(w, r, e); err != nil {
		log.Errorf("failed to render error: %v", err)
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
