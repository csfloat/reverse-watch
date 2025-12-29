package dto

import (
	"fmt"
	"regexp"
)

type MarketplaceUpdateOptions struct {
	Slug     *string
	Name     *string
	IsActive *bool
}

func (o *MarketplaceUpdateOptions) ToFields() map[string]interface{} {
	fields := make(map[string]interface{})
	if o.Slug != nil {
		fields["slug"] = *o.Slug
	}
	if o.Name != nil {
		fields["name"] = *o.Name
	}
	if o.IsActive != nil {
		fields["is_active"] = *o.IsActive
	}
	return fields
}

func (o *MarketplaceUpdateOptions) Validate() error {
	if o.Slug != nil {
		if len(*o.Slug) <= 0 || len(*o.Slug) > 25 {
			return fmt.Errorf("slug must be between 0 and 25 characters long")
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(*o.Slug) {
			return fmt.Errorf("slug must contain only letters, numbers, and hyphens")
		}
	}
	if o.Name != nil {
		if len(*o.Name) <= 0 || len(*o.Name) > 50 {
			return fmt.Errorf("name must be between 0 and 50 characters long")
		}
	}
	return nil
}
