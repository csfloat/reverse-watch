package dto

import (
	"reflect"
	"testing"

	"reverse-watch/internal/domain/models"
)

func TestMarketplaceUpdateOptions_FieldCoverage(t *testing.T) {
	t.Parallel()

	optsType := reflect.TypeOf((*MarketplaceUpdates)(nil)).Elem()
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

	for i := 0; i < optsType.NumField(); i++ {
		optsField := optsType.Field(i)

		excluded := false
		for _, excludedFieldName := range excludedFieldNames {
			if optsField.Name == excludedFieldName {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		marketplaceField, ok := marketplaceFields[optsField.Name]
		if !ok {
			t.Errorf("MarketplaceUpdates contains non-existent Marketplace field: %s", optsField.Name)
			continue
		}

		if optsField.Type.Kind() != reflect.Ptr {
			t.Errorf("MarketplaceUpdates contains non-pointer field: %s", optsField.Name)
		}

		// Ensure same type
		if optsField.Type.Kind() == reflect.Ptr && marketplaceField.Type != optsField.Type.Elem() {
			t.Errorf("MarketplaceUpdates contains field %q with incorrect type", optsField.Name)
		}
	}
}
