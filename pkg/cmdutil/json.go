package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// JSONFieldInfo describes a single selectable JSON field.
type JSONFieldInfo struct {
	JSONName string
	TypeName string
}

// JSONFlagFields returns the available field names for a type used
// in error messages when --json is given without a value.
func JSONFlagFields[T any](name string) []JSONFieldInfo {
	var v T
	rt := reflect.TypeOf(v)
	return jsonFieldsForType(rt)
}

func jsonFieldsForType(rt reflect.Type) []JSONFieldInfo {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return nil
	}
	var fields []JSONFieldInfo
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		tag, ok := f.Tag.Lookup("json")
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		fields = append(fields, JSONFieldInfo{
			JSONName: name,
			TypeName: f.Type.String(),
		})
	}
	return fields
}

// SelectFields extracts a subset of JSON fields from a struct value.
// Nested fields can be accessed with dot notation (e.g. "user.login").
// Fields that don't exist are silently skipped.
func SelectFields(v any, fields []string) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", v)
	}

	// Build json-tag → field index map
	tagToIdx := make(map[string]int)
	for i := 0; i < rv.Type().NumField(); i++ {
		f := rv.Type().Field(i)
		if !f.IsExported() {
			continue
		}
		tag, ok := f.Tag.Lookup("json")
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		tagToIdx[name] = i
	}

	result := make(map[string]any, len(fields))
	for _, fieldPath := range fields {
		parts := strings.SplitN(fieldPath, ".", 2)
		idx, ok := tagToIdx[parts[0]]
		if !ok {
			continue
		}
		fieldVal := rv.Field(idx).Interface()

		if len(parts) == 1 {
			result[parts[0]] = fieldVal
		} else {
			// Nested field access (e.g. "user.login")
			subMap, err := SelectFields(fieldVal, []string{parts[1]})
			if err != nil {
				// Skip nested fields that can't be resolved
				continue
			}
			if existing, ok := result[parts[0]]; ok {
				if em, ok := existing.(map[string]any); ok {
					for k, v := range subMap {
						em[k] = v
					}
				}
			} else {
				result[parts[0]] = subMap
			}
		}
	}
	return result, nil
}

// WriteJSON writes a single value as JSON.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// ParseJSONFlag parses a --json flag value.
// If value is empty, returns nil, false, false (no JSON output).
// If value is "*", returns nil, true, false (full JSON output).
// If value is "list" or "fields", returns nil, false, true (list fields mode).
// Otherwise splits by comma and returns the field list, false, false.
func ParseJSONFlag(value string) (fields []string, full bool, listFields bool) {
	if value == "" {
		return nil, false, false
	}
	if value == "*" {
		return nil, true, false
	}
	if value == "list" || value == "fields" {
		return nil, false, true
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts, false, false
}

// PrintJSONFieldList writes the available JSON fields for type T to w.
func PrintJSONFieldList[T any](w io.Writer) {
	fields := JSONFlagFields[T]("")
	maxLen := 0
	for _, f := range fields {
		if len(f.JSONName) > maxLen {
			maxLen = len(f.JSONName)
		}
	}
	fmt.Fprintf(w, "Available fields:\n")
	for _, f := range fields {
		fmt.Fprintf(w, "  %-*s  %s\n", maxLen, f.JSONName, f.TypeName)
	}
}

// WriteJSONFields writes a slice of structs, selecting only the specified fields.
// If fields is empty, the full struct is serialized.
func WriteJSONFields[T any](w io.Writer, items []T, fields []string) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	if len(fields) == 0 {
		return enc.Encode(items)
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, err := SelectFields(item, fields)
		if err != nil {
			return err
		}
		result = append(result, m)
	}
	return enc.Encode(result)
}

// JSONFlagHelp returns a formatted help string for --json flag's available fields.
func JSONFlagHelp[T any]() string {
	fields := JSONFlagFields[T]("")
	maxLen := 0
	for _, f := range fields {
		if len(f.JSONName) > maxLen {
			maxLen = len(f.JSONName)
		}
	}
	var sb strings.Builder
	sb.WriteString("Output as JSON (use --json=field1,field2 to select specific fields)\n")
	sb.WriteString("Available fields:\n")
	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("  %-*s  %s\n", maxLen, f.JSONName, f.TypeName))
	}
	return sb.String()
}

// StatusResult is a simple success/failure response.
type StatusResult struct {
	Success bool `json:"success"`
}
