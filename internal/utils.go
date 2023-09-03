package internal

import (
	"reflect"
	"strings"
)

func IsModified(old, newV interface{}, field string) bool {
	o := reflect.ValueOf(old)
	n := reflect.ValueOf(newV)

	parts := strings.Split(field, ".")

	if len(parts) == 1 {
		of := o.FieldByName(field)
		nf := n.FieldByName(field)
		return of.IsValid() && nf.IsValid() &&
			!reflect.DeepEqual(of.Interface(), nf.Interface())
	} else {
		of := o.FieldByName(parts[0])
		nf := n.FieldByName(parts[0])

		if !of.IsValid() || !nf.IsValid() {
			return false
		}

		return IsModified(of.Interface(), nf.Interface(), strings.Join(parts[1:], "."))
	}
}
