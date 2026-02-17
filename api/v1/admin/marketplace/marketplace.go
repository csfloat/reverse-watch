package marketplace

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
			return err
		}

		var err error
		rawKey, err = tx.Key().Create(marketplace.Slug, models.PermissionManage)
		if err != nil {
			return err
		}

		audit := models.NewMarketplaceAdminAudit(models.TargetActionAddMarketplace, marketplace.Slug, nil)
		if err := tx.AdminAudit().Create(audit); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logging.Log.Errorf("failed to onboard marketplace: %v", err)
		render.Error(w, r, err)
		return
	}

	storedMarketplace, err = factory.Marketplace().Read(marketplace.Slug)
	if err != nil {
		logging.Log.Errorf("failed to read newly onboarded marketplace: %v", err)
		render.Error(w, r, err)
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
			return err
		}

		var details *models.RawJsonb
		var err error
		details, err = models.ToRawJsonb(opts)
		if err != nil {
			return err
		}

		audit := models.NewMarketplaceAdminAudit(models.TargetActionUpdateMarketplace, slug, details)
		if err := tx.AdminAudit().Create(audit); err != nil {
			return err
		}

		updatedMarketplace, err = tx.Marketplace().Read(slug)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logging.Log.Errorf("failed to update marketplace: %v", err)
		render.Error(w, r, err)
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
