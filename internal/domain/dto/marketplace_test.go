package dto

import (
	"reflect"
	"slices"
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/util"
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

func TestMarketplaceUpdates_Validate(t *testing.T) {
	t.Parallel()

	updates := &MarketplaceUpdates{
		Name:     util.Ptr("Test Marketplace"),
		IsActive: util.Ptr(true),
	}

	if err := updates.Validate(); err != nil {
		t.Errorf("Validate(): %v", err)
	}
}

func TestMarketplaceUpdates_Validate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		updates *MarketplaceUpdates
		wantErr string
	}{
		{
			name:    "emptyUpdates",
			updates: &MarketplaceUpdates{},
			wantErr: "marketplace updates must have at least one field",
		},
		{
			name: "emptyName",
			updates: &MarketplaceUpdates{
				Name: util.Ptr(""),
			},
			wantErr: "name must be between 1 and 50 characters long",
		},
		{
			name: "nameTooLong",
			updates: &MarketplaceUpdates{
				Name: util.Ptr("Test Marketplace this name is invalid since it is too long"),
			},
			wantErr: "name must be between 1 and 50 characters long",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.updates.Validate()
			if err == nil {
				t.Errorf("Validate(): got nil error, want %s", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Validate(): got error %s, want %s", err.Error(), tc.wantErr)
			}
		})
	}
}
