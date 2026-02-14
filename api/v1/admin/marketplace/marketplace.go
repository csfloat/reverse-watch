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

	storedMarketplace, err = factory.Marketplace().Read(marketplace.Slug)
	if err != nil {
		render.Errorf(w, r, errors.DBRead, "failed to read marketplace %q", marketplace.Slug)
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
