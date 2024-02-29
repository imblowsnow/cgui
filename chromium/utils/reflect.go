package utils

import "reflect"

func GetObjField(obj interface{}, fieldName string) interface{} {
	reflectValue := reflect.ValueOf(obj)
	// 判断是否是指针
	if reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	fieldValue := reflectValue.FieldByName(fieldName)
	return fieldValue.Interface()
}
