package gmongo

import "reflect"

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
		tag := v.Field(i).Tag.Get(tag)

		field := reflectValue.Field(i).Interface()
		if tag != "" && tag != "-" {
			if v.Field(i).Type.Kind() == reflect.Struct {
				res[tag] = structToMapWithTags(field, tag)
			} else {
				res[tag] = field
			}
		}
	}
	return res
}
