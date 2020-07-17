package utils

import (
	"fmt"
	"github.com/fatih/structs"
	"reflect"
)

type Map map[string]string

func StructToMap(v interface{}) map[string]string {

	map_v := structs.Map(v)
	flat_v := Flatten(map_v)

	return (map[string]string)(flat_v)
}

// Flatten takes a structure and turns into a flat map[string]string.
//
// Within the "thing" parameter, only primitive values are allowed. Structs are
// not supported. Therefore, it can only be slices, maps, primitives, and
// any combination of those together.
//
// See the tests for examples of what inputs are turned into.
func Flatten(thing map[string]interface{}) Map {
	result := make(map[string]string)

	for k, raw := range thing {
		flatten(result, k, reflect.ValueOf(raw))
	}

	return Map(result)
}

func flatten(result map[string]string, prefix string, v reflect.Value) {
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}

		v = v.Elem()
		flatten(result, prefix, v)
		return
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			result[prefix] = "true"
		} else {
			result[prefix] = "false"
		}
	case reflect.Int, reflect.Int32, reflect.Int64:
		result[prefix] = fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		result[prefix] = fmt.Sprintf("%d", v.Uint())
	case reflect.Map:
		flattenMap(result, prefix, v)
	case reflect.Slice:
		flattenSlice(result, prefix, v)
	case reflect.String:
		result[prefix] = v.String()
	default:
		panic(fmt.Sprintf("Unknown: %s", v))
	}
}

func flattenMap(result map[string]string, prefix string, v reflect.Value) {
	for _, k := range v.MapKeys() {
		if k.Kind() == reflect.Interface {
			k = k.Elem()
		}

		if k.Kind() != reflect.String {
			panic(fmt.Sprintf("%s: map key is not string: %s", prefix, k))
		}

		flatten(result, fmt.Sprintf("%s.%s", prefix, k.String()), v.MapIndex(k))
	}
}

func flattenSlice(result map[string]string, prefix string, v reflect.Value) {
	//prefix = prefix + "."

	//result[prefix+"#"] = fmt.Sprintf("%d", v.Len())
	for i := 0; i < v.Len(); i++ {
		flatten(result, fmt.Sprintf("%s[%d]", prefix, i), v.Index(i))
	}
}
