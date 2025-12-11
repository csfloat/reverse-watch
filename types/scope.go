package types

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type Permission uint8

const (
	PermissionRead   Permission = 1
	PermissionWrite  Permission = 1 << 1
	PermissionManage Permission = 1 << 2
	PermissionAdmin  Permission = 1 << 3
)

type Scope Permission

const (
	ScopeRead   = Scope(PermissionRead)
	ScopeWrite  = Scope(PermissionRead | PermissionWrite)
	ScopeManage = Scope(PermissionRead | PermissionWrite | PermissionManage)
	ScopeAdmin  = Scope(PermissionRead | PermissionWrite | PermissionManage | PermissionAdmin)
)

var ScopeToName = map[Scope]string{
	ScopeRead:   "read",
	ScopeWrite:  "write",
	ScopeManage: "manage",
	ScopeAdmin:  "admin",
}

func (s Scope) MarshalJSON() ([]byte, error) {
	str, ok := ScopeToName[s]
	if !ok {
		return nil, fmt.Errorf("invalid scope")
	}
	return json.Marshal(str)
}

func (s *Scope) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	for scope, name := range ScopeToName {
		if str == name {
			*s = scope
		}
	}
	return nil
}

func (s Scope) String() string {
	if v, ok := ScopeToName[s]; ok {
		return v
	}
	return "invalid"
}

func (s Scope) HasPermission(p Permission) bool {
	return uint8(s)&uint8(p) == uint8(p)
}

func (s Scope) IsValid() bool {
	switch s {
	case ScopeRead, ScopeWrite, ScopeManage, ScopeAdmin:
		return true
	default:
		return false
	}
}

type ScopeEnum struct {
	Scope Scope  `gorm:"primaryKey;autoIncrement:false"`
	Name  string `gorm:"unique"`
}

func (s *ScopeEnum) BeforeCreate(tx *gorm.DB) error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	if s.Scope == 0 {
		return fmt.Errorf("scope is required")
	}
	return nil
}
