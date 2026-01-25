package dto

import (
	"fmt"

	"reverse-watch/domain/models"
)

type MarketplaceUpdates struct {
	Name     *string
	IsActive *bool
}

func (u *MarketplaceUpdates) ToFields() map[string]interface{} {
	fields := make(map[string]interface{})
	if u.Name != nil {
		fields["name"] = *u.Name
	}
	if u.IsActive != nil {
		fields["is_active"] = *u.IsActive
	}
	return fields
}

func (u *MarketplaceUpdates) Validate() error {
	if len(u.ToFields()) == 0 {
		return fmt.Errorf("marketplace updates must have at least one field")
	}
	if u.Name != nil {
		if len(*u.Name) <= 0 || len(*u.Name) > 50 {
			return fmt.Errorf("name must be between 1 and 50 characters long")
		}
	}
	return nil
}

func (u *MarketplaceUpdates) AuditDetails() (*models.RawJsonb, error) {
	return models.ToRawJsonb(u.ToFields())
}
