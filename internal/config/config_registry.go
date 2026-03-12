package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type FieldType int

const (
	TypeString FieldType = iota
	TypeInt
	TypeUint64
	TypeBool
	TypeStringSlice
)

type ConfigField struct {
	Path      string
	Type      FieldType
	Parent    string
	FieldName string
}

var configFieldRegistry []ConfigField

func init() {
	configFieldRegistry = buildFieldRegistry()
}

func buildFieldRegistry() []ConfigField {
	var fields []ConfigField
	cfg := &Config{}

	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		sectionField := t.Field(i)

		// Skip non-struct fields at the top level (like DemoMode)
		if sectionField.Type.Kind() != reflect.Struct {
			continue
		}

		sectionName := strings.ToLower(sectionField.Name)
		sectionVal := v.Field(i)

		for j := 0; j < sectionVal.Type().NumField(); j++ {
			field := sectionVal.Field(j)
			fieldInfo := sectionVal.Type().Field(j)
			fieldName := fieldInfo.Name
			jsonTag := fieldInfo.Tag.Get("json")

			fieldPath := sectionName + "." + strings.Split(jsonTag, ",")[0]

			var fieldType FieldType
			switch field.Kind() {
			case reflect.String:
				fieldType = TypeString
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldType = TypeInt
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldType = TypeUint64
			case reflect.Bool:
				fieldType = TypeBool
			case reflect.Slice:
				if field.Type().Elem().Kind() == reflect.String {
					fieldType = TypeStringSlice
				}
			}

			fields = append(fields, ConfigField{
				Path:      fieldPath,
				Type:      fieldType,
				Parent:    sectionName,
				FieldName: fieldName,
			})
		}
	}

	return fields
}

func GetConfigField(cfg *Config, key string) (any, error) {
	for _, f := range configFieldRegistry {
		if f.Path == key {
			return getFieldValue(cfg, f.Parent, f.FieldName)
		}
	}
	return nil, fmt.Errorf("unknown key %q", key)
}

func SetConfigField(cfg *Config, key string, value string) error {
	for _, f := range configFieldRegistry {
		if f.Path == key {
			return setFieldValue(cfg, f.Parent, f.FieldName, f.Type, value)
		}
	}
	return fmt.Errorf("unknown key %q", key)
}

func getFieldValue(cfg *Config, parent, fieldName string) (any, error) {
	var section reflect.Value

	switch parent {
	case "rpc":
		section = reflect.ValueOf(&cfg.RPC).Elem()
	case "provider":
		section = reflect.ValueOf(&cfg.Provider).Elem()
	case "client":
		section = reflect.ValueOf(&cfg.Client).Elem()
	case "security":
		section = reflect.ValueOf(&cfg.Security).Elem()
	case "logging":
		section = reflect.ValueOf(&cfg.Logging).Elem()
	default:
		return nil, fmt.Errorf("unknown section %q", parent)
	}

	field := section.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("field %q not found in %q", fieldName, parent)
	}

	switch field.Kind() {
	case reflect.String:
		return field.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(field.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint(), nil
	case reflect.Bool:
		return field.Bool(), nil
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			return field.Interface().([]string), nil
		}
	}
	return nil, fmt.Errorf("unsupported field type %v", field.Kind())
}

func setFieldValue(cfg *Config, parent, fieldName string, fieldType FieldType, value string) error {
	var section reflect.Value

	switch parent {
	case "rpc":
		section = reflect.ValueOf(&cfg.RPC).Elem()
	case "provider":
		section = reflect.ValueOf(&cfg.Provider).Elem()
	case "client":
		section = reflect.ValueOf(&cfg.Client).Elem()
	case "security":
		section = reflect.ValueOf(&cfg.Security).Elem()
	case "logging":
		section = reflect.ValueOf(&cfg.Logging).Elem()
	default:
		return fmt.Errorf("unknown section %q", parent)
	}

	field := section.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("field %q not found in %q", fieldName, parent)
	}

	switch fieldType {
	case TypeString:
		field.SetString(value)
	case TypeInt:
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer value %q: %w", value, err)
		}
		field.SetInt(int64(v))
	case TypeUint64:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint64 value %q: %w", value, err)
		}
		field.SetUint(v)
	case TypeBool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool value %q: %w", value, err)
		}
		field.SetBool(v)
	case TypeStringSlice:
		vals := strings.Split(value, ",")
		for i := range vals {
			vals[i] = strings.TrimSpace(vals[i])
		}
		field.Set(reflect.ValueOf(vals))
	default:
		return fmt.Errorf("unsupported field type %v", fieldType)
	}

	return nil
}

func ListConfigFields() []string {
	fields := make([]string, len(configFieldRegistry))
	for i, f := range configFieldRegistry {
		fields[i] = f.Path
	}
	return fields
}
