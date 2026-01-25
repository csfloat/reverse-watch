package dto

import (
	"reflect"
	"slices"
	"testing"

	"reverse-watch/domain/models"
)

func TestMarketplaceUpdates_FieldCoverage(t *testing.T) {
	t.Parallel()

	updatesType := reflect.TypeOf((*MarketplaceUpdates)(nil)).Elem()
	marketplaceType := reflect.TypeOf((*models.Marketplace)(nil)).Elem()

	excludedFieldNames := []string{
		"CreatedAt",
		"UpdatedAt",
	}

	marketplaceFields := make(map[string]reflect.StructField)
	for i := 0; i < marketplaceType.NumField(); i++ {
		field := marketplaceType.Field(i)
		marketplaceFields[field.Name] = field
	}

	for i := 0; i < updatesType.NumField(); i++ {
		updatesField := updatesType.Field(i)

		// Skip excluded fields
		if slices.Contains(excludedFieldNames, updatesField.Name) {
			continue
		}

		marketplaceField, ok := marketplaceFields[updatesField.Name]
		if !ok {
			t.Errorf("MarketplaceUpdates contains non-existent Marketplace field: %s", updatesField.Name)
			continue
		}

		if updatesField.Type.Kind() != reflect.Ptr {
			t.Errorf("MarketplaceUpdates contains non-pointer field: %s", updatesField.Name)
		}

		// Ensure same type
		if updatesField.Type.Kind() == reflect.Ptr && marketplaceField.Type != updatesField.Type.Elem() {
			t.Errorf("MarketplaceUpdates contains field %q with incorrect type", updatesField.Name)
		}
	}
}
