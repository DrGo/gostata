package gostata

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Field holds the extracted information for a struct field.
type Field struct {
	Name      string      // Name from tag "name" or lowercase field name.
	FieldType byte        // Byte code representing the Stata type.
	Label     string      // From tag "label" or defaults to Name.
	Format    string      // Optional format string.
	data      interface{} // The field’s value.
}

// parseStataTag splits a tag string into a map.
func parseStataTag(tag string) map[string]string {
	m := make(map[string]string)
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			m[key] = val
		}
	}
	return m
}

// convertTyp converts a string representing a Stata type to a byte.
func convertTyp(typStr string) (byte, error) {
	if strings.HasPrefix(typStr, "str") {
		numStr := strings.TrimPrefix(typStr, "str")
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid string type: %s", typStr)
		}
		if n < 1 || n > 244 {
			return 0, fmt.Errorf("string type out of range: %s", typStr)
		}
		return byte(n), nil
	}
	switch typStr {
	case "byte":
		return 251, nil
	case "int":
		return 252, nil
	case "long":
		return 253, nil
	case "float":
		return 254, nil
	case "double":
		return 255, nil
	default:
		return 0, fmt.Errorf("unknown type: %s", typStr)
	}
}

// goTypeToStataType maps Go types to Stata type strings.
func goTypeToStataType(t reflect.Type) (string, error) {
	switch t.Kind() {
	case reflect.Int8:
		return "byte", nil
	case reflect.Int16:
		return "int", nil
	case reflect.Int32, reflect.Int64:
		return "long", nil
	case reflect.Float32:
		return "float", nil
	case reflect.Float64:
		return "double", nil
	case reflect.String:
		return "", fmt.Errorf("string type requires explicit 'typ' tag with strN")
	default:
		return "", fmt.Errorf("unsupported Go type %v for Stata type inference", t.Kind())
	}
}

// ExtractFields extracts fields with 'stata' tags from a struct.
func ExtractFields(v interface{}) ([]*Field, error) {
	var fields []*Field

	rt := reflect.TypeOf(v)
	rv := reflect.ValueOf(v)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rv = rv.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return nil, errors.New("ExtractFields: not a struct")
	}

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		tagStr := sf.Tag.Get("stata")
		// if tagStr == "" {
		// 	continue
		// }
		tagMap := parseStataTag(tagStr)

		name := tagMap["name"]
		if name == "" {
			name = strings.ToLower(sf.Name)
		}

		label := tagMap["label"]
		if label == "" {
			label = name
		}

		var typStr string
		var err error
		if t, ok := tagMap["typ"]; ok && t != "" {
			typStr = t
		} else {
			typStr, err = goTypeToStataType(sf.Type)
			if err != nil {
				return nil, fmt.Errorf("field %s: %v", sf.Name, err)
			}
		}

		fieldType, err := convertTyp(typStr)
		if err != nil {
			return nil, fmt.Errorf("field %s: %v", sf.Name, err)
		}

		format := tagMap["format"]

		fields = append(fields, &Field{
			Name:      name,
			FieldType: fieldType,
			Label:     label,
			Format:    format,
			data:   rv.Field(i).Interface(),
		})
	}

	if len(fields) == 0 {
		return nil, errors.New("ExtractFields: no fields found")
	}
	return fields, nil
}

func calcRecordSize(fields []*Field) int {
        recordSize := 0
        for _, f := range fields {
                switch f.FieldType {
                case StataByteId:
                        recordSize++
                case StataIntId:
                        recordSize += 2
                case StataLongId:
                        recordSize += 4
                case StataFloatId:
                        recordSize += 4
                case StataDoubleId:
                        recordSize += 8
                default: // String type
                        recordSize += int(f.FieldType)
                }
        }
        return recordSize
}
