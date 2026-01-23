package dto

import (
	"reflect"
	"slices"
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/util"

	"github.com/google/go-cmp/cmp"
)

func TestReversalUpdateOptions_FieldCoverage(t *testing.T) {
	t.Parallel()

	optsType := reflect.TypeOf((*ReversalUpdateOptions)(nil)).Elem()
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

	for i := 0; i < optsType.NumField(); i++ {
		optsField := optsType.Field(i)

		// Skip excluded fields
		if slices.Contains(excludedFieldNames, optsField.Name) {
			continue
		}

		reversalField, ok := reversalFields[optsField.Name]
		if !ok {
			t.Errorf("ReversalUpdateOptions contains non-existent Reversal field: %s", optsField.Name)
			continue
		}

		if optsField.Type.Kind() != reflect.Ptr {
			t.Errorf("ReversalUpdateOptions contains non-pointer field: %s", optsField.Name)
		}

		// Ensure same type
		if reversalField.Type.Kind() == reflect.Ptr {
			if optsField.Type != reversalField.Type {
				t.Errorf("got type %v for field %q, wanted %v", optsField.Type, optsField.Name, reversalField.Type)
			}
		} else {
			if optsField.Type.Elem() != reversalField.Type {
				t.Errorf("got type %v for field %q, wanted %v", optsField.Type.Elem(), optsField.Name, reversalField.Type)
			}
		}
	}
}

func TestReversalUpdateOptions_ToFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts *ReversalUpdateOptions
		want map[string]interface{}
	}{
		{
			name: "allFields",
			opts: &ReversalUpdateOptions{
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
			opts: &ReversalUpdateOptions{
				Source: util.Ptr(models.SourceDirect),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceDirect),
				"related_steam_id": nil,
			},
		},
		{
			name: "sourceRelatedUser",
			opts: &ReversalUpdateOptions{
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
			opts: &ReversalUpdateOptions{
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
			if diff := cmp.Diff(tc.opts.ToFields(), tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReversalUpdateOptions_Validate(t *testing.T) {
	t.Parallel()

	opts := &ReversalUpdateOptions{
		Source:         util.Ptr(models.SourceRelatedUser),
		RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
		ReversedAt:     util.Ptr(models.Epoch + 1),
		ExpungedAt:     util.Ptr(models.Epoch + 1),
	}

	if err := opts.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestReversalUpdateOptions_Validate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		opts    *ReversalUpdateOptions
		wantErr string
	}{
		{
			name: "invalidSourceAndRelatedSteamID",
			opts: &ReversalUpdateOptions{
				Source:         util.Ptr(models.SourceDirect),
				RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
			},
			wantErr: "invalid related_steam_id and source combination",
		},
		{
			name: "invalidReversedAt",
			opts: &ReversalUpdateOptions{
				ReversedAt: util.Ptr(uint64(0)),
			},
			wantErr: "reversed_at is invalid",
		},
		{
			name: "reversedAtInFuture",
			opts: &ReversalUpdateOptions{
				ReversedAt: util.Ptr(uint64(99999999999999999)),
			},
			wantErr: "reversed_at is invalid",
		},
		{
			name: "invalidExpungedAt",
			opts: &ReversalUpdateOptions{
				ExpungedAt: util.Ptr(uint64(0)),
			},
			wantErr: "expunged_at is invalid",
		},
		{
			name: "expungedAtInFuture",
			opts: &ReversalUpdateOptions{
				ExpungedAt: util.Ptr(uint64(99999999999999999)),
			},
			wantErr: "expunged_at is invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()
			if err == nil {
				t.Fatalf("Validate(): got nil error, wanted %s", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Validate(): got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}
