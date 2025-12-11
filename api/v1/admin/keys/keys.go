package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
	"reverse-watch/services/private"
	"reverse-watch/types"
)

func adminCreateKeyHandler(w http.ResponseWriter, r *http.Request) {
	privateSvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)

	var req struct {
		MarketplaceSlug string      `json:"marketplace_slug"`
		Scope           types.Scope `json:"scope"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.MarketplaceSlug == "" || !req.Scope.IsValid() {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	rawKey, err := privateSvc.NewKey(req.MarketplaceSlug, req.Scope)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	render.JSON(w, r, rawKey)
}
