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
		key := v.Field(i).Tag.Get(tag)

		// remove omitempty
		key = removeStringFromStringIfExists(key, ",omitempty")

		field := reflectValue.Field(i).Interface()
		if key != "" && key != "-" {
			if v.Field(i).Type.Kind() == reflect.Struct {
				res[key] = structToMapWithTags(field, tag)
			} else {
				res[key] = field
			}
		}
	}
	return res
}
