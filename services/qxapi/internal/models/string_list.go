package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringList is stored as JSON on launcher_instances.
type StringList []string

func (l StringList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (l *StringList) Scan(value any) error {
	if value == nil {
		*l = StringList{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported string list type %T", value)
	}
	if len(data) == 0 {
		*l = StringList{}
		return nil
	}
	return json.Unmarshal(data, l)
}
