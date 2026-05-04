package gmongo

import (
	"reflect"
	"strings"
)

func removeStringFromStringIfExists(str string, remove string) string {
	// first check length
	if len(str) < len(remove) {
		return str
	}

	// Use strings.Replace to remove the substring
	return strings.Replace(str, remove, "", -1)
}

func parseTag(rawTag string) (name string, opts []string) {
	parts := strings.Split(rawTag, ",")
	return parts[0], parts[1:]
}

func hasTagOpt(opts []string, opt string) bool {
	for _, o := range opts {
		if o == opt {
			return true
		}
	}
	return false
}

func structToMapWithTags(obj interface{}, tag string) map[string]interface{} {
	res := map[string]interface{}{}
	if obj == nil {
		return res
	}
	v := reflect.TypeOf(obj)
	reflectValue := reflect.ValueOf(obj)
	reflectValue = reflect.Indirect(reflectValue)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		rawTag := v.Field(i).Tag.Get(tag)
		name, opts := parseTag(rawTag)

		fieldType := v.Field(i).Type
		fieldVal := reflectValue.Field(i)

		if hasTagOpt(opts, "inline") {
			innerType := fieldType
			innerVal := fieldVal
			if innerType.Kind() == reflect.Ptr {
				if innerVal.IsNil() {
					continue
				}
				innerType = innerType.Elem()
				innerVal = innerVal.Elem()
			}
			if innerType.Kind() != reflect.Struct {
				continue
			}
			sub := structToMapWithTags(innerVal.Interface(), tag)
			for k, val := range sub {
				res[k] = val
			}
			continue
		}

		if name == "" || name == "-" {
			continue
		}

		field := fieldVal.Interface()
		if fieldType.Kind() == reflect.Struct {
			res[name] = structToMapWithTags(field, tag)
		} else {
			res[name] = field
		}
	}
	return res
}
