package dto

import (
	"reflect"
	"slices"
	"testing"

	"reverse-watch/domain/models"
	"reverse-watch/util"

	"github.com/google/go-cmp/cmp"
)

func TestReversalUpdates_FieldCoverage(t *testing.T) {
	t.Parallel()

	updatesType := reflect.TypeOf((*ReversalUpdates)(nil)).Elem()
	reversalType := reflect.TypeOf((*models.Reversal)(nil)).Elem()

	excludedFieldNames := []string{
		"CreatedAt",
		"UpdatedAt",
		"DeletedAt",
	}

	reversalFields := make(map[string]reflect.StructField)
	for i := 0; i < reversalType.NumField(); i++ {
		field := reversalType.Field(i)
		reversalFields[field.Name] = field
	}

	for i := 0; i < updatesType.NumField(); i++ {
		updatesField := updatesType.Field(i)

		// Skip excluded fields
		if slices.Contains(excludedFieldNames, updatesField.Name) {
			continue
		}

		reversalField, ok := reversalFields[updatesField.Name]
		if !ok {
			t.Errorf("ReversalUpdates contains non-existent Reversal field: %s", updatesField.Name)
			continue
		}

		if updatesField.Type.Kind() != reflect.Ptr {
			t.Errorf("ReversalUpdates contains non-pointer field: %s", updatesField.Name)
		}

		// Ensure same type
		if reversalField.Type.Kind() == reflect.Ptr {
			if updatesField.Type != reversalField.Type {
				t.Errorf("got type %v for field %q, wanted %v", updatesField.Type, updatesField.Name, reversalField.Type)
			}
		} else {
			if updatesField.Type.Elem() != reversalField.Type {
				t.Errorf("got type %v for field %q, wanted %v", updatesField.Type.Elem(), updatesField.Name, reversalField.Type)
			}
		}
	}
}

func TestReversalUpdates_ToFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		updates *ReversalUpdates
		want    map[string]interface{}
	}{
		{
			name: "allFields",
			updates: &ReversalUpdates{
				Source:         util.Ptr(models.SourceRelatedUser),
				RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
				ReversedAt:     util.Ptr(models.Epoch + 1),
				ExpungedAt:     util.Ptr(models.Epoch + 1),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceRelatedUser),
				"related_steam_id": util.Ptr(models.SteamID(76561197960287931)),
				"reversed_at":      util.Ptr(models.Epoch + 1),
				"expunged_at":      util.Ptr(models.Epoch + 1),
			},
		},
		{
			name: "sourceDirect",
			updates: &ReversalUpdates{
				Source: util.Ptr(models.SourceDirect),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceDirect),
				"related_steam_id": nil,
			},
		},
		{
			name: "sourceRelatedUser",
			updates: &ReversalUpdates{
				Source:         util.Ptr(models.SourceRelatedUser),
				RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceRelatedUser),
				"related_steam_id": util.Ptr(models.SteamID(76561197960287931)),
			},
		},
		{
			name: "sourceUserReport",
			updates: &ReversalUpdates{
				Source: util.Ptr(models.SourceUserReport),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceUserReport),
				"related_steam_id": nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.updates.ToFields(), tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReversalUpdates_Validate(t *testing.T) {
	t.Parallel()

	updates := &ReversalUpdates{
		Source:         util.Ptr(models.SourceRelatedUser),
		RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
		ReversedAt:     util.Ptr(models.Epoch + 1),
		ExpungedAt:     util.Ptr(models.Epoch + 1),
	}

	if err := updates.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestReversalUpdates_Validate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		updates *ReversalUpdates
		wantErr string
	}{
		{
			name:    "nilUpdates",
			updates: nil,
			wantErr: "reversal updates cannot be nil",
		},
		{
			name:    "emptyUpdates",
			updates: &ReversalUpdates{},
			wantErr: "reversal updates must have at least one field",
		},
		{
			name: "invalidSourceAndRelatedSteamID",
			updates: &ReversalUpdates{
				Source:         util.Ptr(models.SourceDirect),
				RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
			},
			wantErr: "invalid related_steam_id and source combination",
		},
		{
			name: "invalidReversedAt",
			updates: &ReversalUpdates{
				ReversedAt: util.Ptr(uint64(0)),
			},
			wantErr: "reversed_at is invalid",
		},
		{
			name: "reversedAtInFuture",
			updates: &ReversalUpdates{
				ReversedAt: util.Ptr(uint64(99999999999999999)),
			},
			wantErr: "reversed_at is invalid",
		},
		{
			name: "invalidExpungedAt",
			updates: &ReversalUpdates{
				ExpungedAt: util.Ptr(uint64(0)),
			},
			wantErr: "expunged_at is invalid",
		},
		{
			name: "expungedAtInFuture",
			updates: &ReversalUpdates{
				ExpungedAt: util.Ptr(uint64(99999999999999999)),
			},
			wantErr: "expunged_at is invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.updates.Validate()
			if err == nil {
				t.Fatalf("Validate(): got nil error, wanted %s", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Validate(): got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}
