package marketplace

import (
	"encoding/json"
	"net/http"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
)

type onboardMarketplaceRequest struct {
	MarketplaceSlug string `json:"marketplace_slug"`
	Name            string `json:"name"`
}

func onboardMarketplace(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	var req onboardMarketplaceRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.MarketplaceSlug == "" || req.Name == "" {
		render.Errorf(w, r, errors.BadRequest, "fields cannot be empty")
		return
	}

	marketplace := &models.Marketplace{
		Slug:     req.MarketplaceSlug,
		Name:     req.Name,
		IsActive: true,
	}

	var rawKey *dto.RawKey
	var storedMarketplace *models.Marketplace

	err := factory.RunInTransactionPrivate(func(tx repository.PrivateTransaction) error {
		if err := tx.Marketplace().Create(marketplace); err != nil {
			return errors.New(errors.DBCreate, "failed to create marketplace")
		}

		var err error
		rawKey, err = tx.Key().Create(marketplace.Slug, models.PermissionManage)
		if err != nil {
			return errors.Newf(errors.DBCreate, err, "failed to create key for marketplace %q", marketplace.Slug)
		}

		storedMarketplace, err = tx.Marketplace().Read(marketplace.Slug)
		if err != nil {
			return errors.Newf(errors.DBRead, err, "failed to fetch newly created marketplace %q", marketplace.Slug)
		}

		audit := models.NewMarketplaceAdminAudit(models.TargetActionAddMarketplace, marketplace.Slug, nil)
		if err := tx.AdminAudit().Create(audit); err != nil {
			return errors.New(errors.DBCreate, "failed to create admin audit")
		}
		return nil
	})
	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			render.Error(w, r, e)
			return
		}
		render.Errorf(w, r, errors.DBCreate, "failed to create marketplace %q", marketplace.Slug)
		return
	}

	render.JSON(w, r, struct {
		Marketplace *models.Marketplace `json:"marketplace"`
		Key         *dto.RawKey         `json:"key"`
	}{
		Marketplace: storedMarketplace,
		Key:         rawKey,
	})
}

func updateMarketplace(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		render.Errorf(w, r, errors.BadRequest, "slug cannot be empty")
		return
	}

	var opts dto.MarketplaceUpdates
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	var updatedMarketplace *models.Marketplace
	err := factory.RunInTransactionPrivate(func(tx repository.PrivateTransaction) error {
		if err := tx.Marketplace().Update(slug, &opts); err != nil {
			return errors.New(errors.DBUpdate, "failed to update marketplace")
		}

		var details *models.RawJsonb
		var err error
		details, err = models.ToRawJsonb(opts)
		if err != nil {
			return errors.New(errors.InternalServerError, "failed to convert to raw jsonb")
		}

		audit := models.NewMarketplaceAdminAudit(models.TargetActionUpdateMarketplace, slug, details)
		if err := tx.AdminAudit().Create(audit); err != nil {
			return errors.New(errors.DBCreate, "failed to create admin audit")
		}

		updatedMarketplace, err = tx.Marketplace().Read(slug)
		if err != nil {
			return errors.New(errors.DBRead, "failed to find marketplace")
		}
		return nil
	})
	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			render.Error(w, r, e)
			return
		}
		render.Errorf(w, r, errors.DBUpdate, "failed to update marketplace")
		return
	}

	render.JSON(w, r, updatedMarketplace)
}

func deleteMarketplace(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		render.Errorf(w, r, errors.BadRequest, "slug cannot be empty")
		return
	}

	err := factory.RunInTransactionPrivate(func(tx repository.PrivateTransaction) error {
		if err := tx.Marketplace().Delete(slug); err != nil {
			return errors.New(errors.DBDelete, "failed to delete marketplace")
		}

		audit := models.NewMarketplaceAdminAudit(models.TargetActionRemoveMarketplace, slug, nil)
		if err := tx.AdminAudit().Create(audit); err != nil {
			return errors.New(errors.DBCreate, "failed to create admin audit")
		}
		return nil
	})
	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			render.Error(w, r, e)
			return
		}
		render.Errorf(w, r, errors.DBDelete, "failed to delete marketplace")
		return
	}
	w.WriteHeader(http.StatusOK)
}
