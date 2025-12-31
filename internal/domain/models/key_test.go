package models

import (
	"testing"

	"reverse-watch/internal/domain/models/constants"
)

func TestKeyRepository_BeforeCreate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		key  *Key
	}{
		{
			name: "validKey",
			key: &Key{
				ID:              "valid-key-id",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "test-marketplace",
				Permissions:     PermissionRead | PermissionWrite,
			},
		},
		{
			name: "adminKeyForCSFloat",
			key: &Key{
				ID:              "admin-key-id",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "csfloat",
				Permissions:     PermissionAdmin,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.key.BeforeCreate(nil); err != nil {
				t.Fatalf("BeforeCreate() error: %v", err)
			}
		})
	}

}

func TestKeyRepository_BeforeCreate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		key     *Key
		wantErr string
	}{
		{
			name: "emptyID",
			key: &Key{
				ID:              "",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "test-marketplace",
				Permissions:     PermissionRead,
			},
			wantErr: "key hash is required",
		},
		{
			name: "emptyEnvironment",
			key: &Key{
				ID:              "test-key-id-2",
				Environment:     "",
				MarketplaceSlug: "test-marketplace",
				Permissions:     PermissionWrite | PermissionManage | PermissionExport,
			},
			wantErr: "environment is required",
		},
		{
			name: "noPermissions",
			key: &Key{
				ID:              "test-key-id-3",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "test-marketplace",
				Permissions:     PermissionNone,
			},
			wantErr: "at least one permission is required",
		},
		{
			name: "adminPermissionNonCSFloat",
			key: &Key{
				ID:              "test-key-id-4",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "test-marketplace",
				Permissions:     PermissionAdmin,
			},
			wantErr: "admin scoped keys can only be created for CSFloat",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.key.BeforeCreate(nil)
			if err == nil {
				t.Fatal("Create(): got nil error, wanted error")
			}

			if err.Error() != tc.wantErr {
				t.Errorf("Create(): got error %q, want %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestKey_HasPermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		key         *Key
		permissions []Permissions
		want        bool
	}{
		{
			name: "hasReadPermission",
			key: &Key{
				Permissions: PermissionRead,
			},
			permissions: []Permissions{PermissionRead},
			want:        true,
		},
		{
			name: "hasMultiplePermissions",
			key: &Key{
				Permissions: PermissionRead | PermissionWrite | PermissionManage,
			},
			permissions: []Permissions{PermissionRead, PermissionWrite, PermissionManage},
			want:        true,
		},
		{
			name: "lacksPermission",
			key: &Key{
				Permissions: PermissionRead | PermissionWrite,
			},
			permissions: []Permissions{PermissionManage},
			want:        false,
		},
		{
			name: "lacksOneOfMultiplePermissions",
			key: &Key{
				Permissions: PermissionRead | PermissionWrite,
			},
			permissions: []Permissions{PermissionRead, PermissionManage},
			want:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.key.HasPermissions(tc.permissions...)
			if got != tc.want {
				t.Errorf("HasPermissions(): got %v, want %v", got, tc.want)
			}
		})
	}
}
