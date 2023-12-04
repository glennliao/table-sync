package tablesync

import (
	"bytes"
	"github.com/glennliao/table-sync/model"
	"unicode"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gstructs"
	"reflect"
	"strings"
)

func parseDdlTag(col model.Column, tag string) model.Column {
	if col.DDLTag == nil {
		col.DDLTag = make(map[string]string)
	}
	for _, str := range strings.Split(tag, ";") {
		x := strings.Split(str, ":")
		switch x[0] {
		case "comment":
			if len(x) > 1 {
				col.Comment = strings.ReplaceAll(strings.Join(x[1:], ":"), "'", "\\'")
			}
		default:
			if x[0] != "" {
				if len(x) > 1 {
					col.DDLTag[x[0]] = strings.Join(x[1:], ":")
				} else {
					col.DDLTag[x[0]] = "true"
				}
			}
		}
	}

	return col
}

// fields retrieves and returns the fields of `pointer` as slice.
func fields(in gstructs.FieldsInput) ([]gstructs.Field, error) {
	var (
		ok                   bool
		fieldFilterMap       = make(map[string]struct{})
		retrievedFields      = make([]gstructs.Field, 0)
		currentLevelFieldMap = make(map[string]gstructs.Field)
	)
	rangeFields, err := getFieldValues(in.Pointer)
	if err != nil {
		return nil, err
	}

	for index := 0; index < len(rangeFields); index++ {
		field := rangeFields[index]
		currentLevelFieldMap[field.Name()] = field
	}

	for index := 0; index < len(rangeFields); index++ {
		field := rangeFields[index]
		if _, ok = fieldFilterMap[field.Name()]; ok {
			continue
		}
		if field.IsEmbedded() {
			if in.RecursiveOption != gstructs.RecursiveOptionNone {
				switch in.RecursiveOption {
				case gstructs.RecursiveOptionEmbeddedNoTag:
					if field.TagStr() != "" {
						break
					}
					fallthrough

				case gstructs.RecursiveOptionEmbedded:
					structFields, err := fields(gstructs.FieldsInput{
						Pointer:         field.Value,
						RecursiveOption: in.RecursiveOption,
					})
					if err != nil {
						return nil, err
					}
					// The current level fields can overwrite the sub-struct fields with the same name.
					for i := 0; i < len(structFields); i++ {
						var (
							structField = structFields[i]
							fieldName   = structField.Name()
						)
						//if _, ok = fieldFilterMap[fieldName]; ok {
						//	continue
						//}
						fieldFilterMap[fieldName] = struct{}{}
						retrievedFields = append(retrievedFields, structField)
						//if v, ok := currentLevelFieldMap[fieldName]; !ok {
						//	retrievedFields = append(retrievedFields, structField)
						//} else {
						//	retrievedFields = append(retrievedFields, v)
						//}
					}
					continue
				}
			}
			continue
		}
		fieldFilterMap[field.Name()] = struct{}{}
		retrievedFields = append(retrievedFields, field)
	}
	return retrievedFields, nil
}

func getFieldValues(value interface{}) ([]gstructs.Field, error) {
	var (
		reflectValue reflect.Value
		reflectKind  reflect.Kind
	)
	if v, ok := value.(reflect.Value); ok {
		reflectValue = v
		reflectKind = reflectValue.Kind()
	} else {
		reflectValue = reflect.ValueOf(value)
		reflectKind = reflectValue.Kind()
	}
	for {
		switch reflectKind {
		case reflect.Ptr:
			if !reflectValue.IsValid() || reflectValue.IsNil() {
				// If pointer is type of *struct and nil, then automatically create a temporary struct.
				reflectValue = reflect.New(reflectValue.Type().Elem()).Elem()
				reflectKind = reflectValue.Kind()
			} else {
				reflectValue = reflectValue.Elem()
				reflectKind = reflectValue.Kind()
			}
		case reflect.Array, reflect.Slice:
			reflectValue = reflect.New(reflectValue.Type().Elem()).Elem()
			reflectKind = reflectValue.Kind()
		default:
			goto exitLoop
		}
	}

exitLoop:
	for reflectKind == reflect.Ptr {
		reflectValue = reflectValue.Elem()
		reflectKind = reflectValue.Kind()
	}
	if reflectKind != reflect.Struct {
		return nil, gerror.NewCode(
			gcode.CodeInvalidParameter,
			"given value should be either type of struct/*struct/[]struct/[]*struct",
		)
	}
	var (
		structType = reflectValue.Type()
		length     = reflectValue.NumField()
		fields     = make([]gstructs.Field, length)
	)
	for i := 0; i < length; i++ {
		fields[i] = gstructs.Field{
			Value: reflectValue.Field(i),
			Field: structType.Field(i),
		}
	}
	return fields, nil
}

func ListEq[T comparable](a, b []T) bool {
	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func convertCamelToUnderScore(str string) string {
	buffer := bytes.NewBufferString("")
	for i, r := range str {
		if unicode.IsUpper(r) {
			if i != 0 {
				if unicode.IsLower(rune2(str, i-1)) || (unicode.IsUpper(rune2(str, i-1)) && i+1 < len(str) && unicode.IsLower(rune2(str, i+1))) {
					buffer.WriteString("_")
				}
			}
		}
		buffer.WriteRune(unicode.ToLower(r))
	}
	return buffer.String()
}

func rune2(s string, index int) rune {
	return []rune(s)[index]
}
