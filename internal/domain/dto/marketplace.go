package dto

import (
	"fmt"

	"reverse-watch/internal/domain/models"
)

type MarketplaceUpdateOptions struct {
	Name     *string
	IsActive *bool
}

func (o *MarketplaceUpdateOptions) ToFields() map[string]interface{} {
	fields := make(map[string]interface{})
	if o.Name != nil {
		fields["name"] = *o.Name
	}
	if o.IsActive != nil {
		fields["is_active"] = *o.IsActive
	}
	return fields
}

func (o *MarketplaceUpdateOptions) Validate() error {
	if o.Name != nil {
		if len(*o.Name) <= 0 || len(*o.Name) > 50 {
			return fmt.Errorf("name must be between 1 and 50 characters long")
		}
	}
	return nil
}

func (o *MarketplaceUpdateOptions) AuditDetails() (*models.RawJsonb, error) {
	return models.ToRawJsonb(o.ToFields())
}
