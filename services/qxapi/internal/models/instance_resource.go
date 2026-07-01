package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// InstanceResourceEntry describes a mod, resource pack, or shader installed on a client instance.
type InstanceResourceEntry struct {
	Source        string `json:"source"`
	ProjectID     string `json:"project_id,omitempty"`
	ProjectName   string `json:"project_name"`
	VersionID     string `json:"version_id,omitempty"`
	VersionNumber string `json:"version_number,omitempty"`
	Filename      string `json:"filename"`
	ResourceType  string `json:"resource_type"`
	InstalledAt   string `json:"installed_at"`
}

// InstanceResourceList is stored as JSON on launcher_instances.
type InstanceResourceList []InstanceResourceEntry

func (l InstanceResourceList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (l *InstanceResourceList) Scan(value any) error {
	if value == nil {
		*l = InstanceResourceList{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported instance resource list type %T", value)
	}
	if len(data) == 0 {
		*l = InstanceResourceList{}
		return nil
	}
	return json.Unmarshal(data, l)
}
