package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/logging"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
)

func createKey(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	var req struct {
		MarketplaceSlug string             `json:"marketplace_slug"`
		Permissions     models.Permissions `json:"permissions"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.MarketplaceSlug == "" || req.Permissions == 0 {
		render.Errorf(w, r, errors.BadRequest, "marketplace slug and permissions are required")
		return
	}

	if req.Permissions.HasPermissions(models.PermissionAdmin) && req.MarketplaceSlug != "csfloat" {
		render.Errorf(w, r, errors.BadRequest, "admin scoped keys can only be created for csfloat")
		return
	}

	var rawKey *dto.RawKey
	err := factory.RunInTransactionPrivate(func(tx repository.PrivateTransaction) error {
		var err error
		rawKey, err = tx.Key().Create(req.MarketplaceSlug, req.Permissions)
		if err != nil {
			return err
		}

		jsonb, err := models.ToRawJsonb(req)
		if err != nil {
			return err
		}

		audit := models.NewKeyAdminAudit(models.TargetActionAddKey, rawKey.ID, jsonb)
		if err := tx.AdminAudit().Create(audit); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logging.Log.Errorf("failed to create key: %v", err)
		render.Error(w, r, err)
		return
	}

	render.JSON(w, r, rawKey)
}

func deleteKey(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	id := chi.URLParam(r, "id")
	if id == "" {
		render.Errorf(w, r, errors.BadRequest, "id must not be empty")
		return
	}

	err := factory.RunInTransactionPrivate(func(tx repository.PrivateTransaction) error {
		key, err := tx.Key().Read(id)
		if err != nil {
			return err
		}

		if err := tx.Key().Delete(id); err != nil {
			return err
		}

		details := &dto.DeleteKeyDetails{
			MarketplaceSlug: key.MarketplaceSlug,
			Permissions:     key.Permissions,
		}

		jsonb, err := models.ToRawJsonb(details)
		if err != nil {
			return err
		}

		audit := models.NewKeyAdminAudit(models.TargetActionRemoveKey, id, jsonb)
		if err := tx.AdminAudit().Create(audit); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logging.Log.Errorf("failed to delete key: %v", err)
		render.Error(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
