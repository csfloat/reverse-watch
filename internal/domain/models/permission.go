package models

import (
	"encoding/json"
	"fmt"
)

// Permissions is a bit-field that encompasses a set of permissions
type Permissions uint32

const (
	PermissionAdmin  Permissions = 1
	PermissionDelete Permissions = 1 << 1
	PermissionManage Permissions = 1 << 2
	PermissionWrite  Permissions = 1 << 3
	PermissionRead   Permissions = 1 << 4
)

func (p *Permissions) HasPermissions(permissions ...Permissions) bool {
	expected := uint32(0)
	for _, permission := range permissions {
		expected |= uint32(permission)
	}
	if expected == 0 {
		return false
	}
	return uint32(*p)&expected == expected
}

func (p *Permissions) AddPermission(permission Permissions) {
	*p = Permissions(uint32(*p) | uint32(permission))
}

func (p *Permissions) RemovePermission(permission Permissions) {
	*p = Permissions(uint32(*p) & ^uint32(permission))
}

func (p *Permissions) MarshalJSON() ([]byte, error) {
	var permissions []string
	if p.HasPermissions(PermissionAdmin) {
		permissions = append(permissions, "admin")
	}
	if p.HasPermissions(PermissionDelete) {
		permissions = append(permissions, "delete")
	}
	if p.HasPermissions(PermissionManage) {
		permissions = append(permissions, "manage")
	}
	if p.HasPermissions(PermissionWrite) {
		permissions = append(permissions, "write")
	}
	if p.HasPermissions(PermissionRead) {
		permissions = append(permissions, "read")
	}
	return json.Marshal(permissions)
}

func (p *Permissions) UnmarshalJSON(data []byte) error {
	var permissions []string
	if err := json.Unmarshal(data, &permissions); err != nil {
		return err
	}

	for _, permission := range permissions {
		switch permission {
		case "admin":
			p.AddPermission(PermissionAdmin)
		case "delete":
			p.AddPermission(PermissionDelete)
		case "manage":
			p.AddPermission(PermissionManage)
		case "write":
			p.AddPermission(PermissionWrite)
		case "read":
			p.AddPermission(PermissionRead)
		default:
			return fmt.Errorf("unknown permission: %s", p)
		}
	}
	return nil
}
