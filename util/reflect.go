package util

import (
	"reflect"
	"regexp"
)

func GetPkgNames(t reflect.Type, pkgNameLength ...int) []string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	packageNames := pkgNameRe.Split(t.PkgPath(), -1)
	if len(pkgNameLength) > 0 {
		l := pkgNameLength[0]
		if len(packageNames) > l {
			packageNames = packageNames[len(packageNames)-l:]
		}
	}
	return packageNames
}

var pkgNameRe = regexp.MustCompile(`[_\-/]+`)

func IsZero(val any) bool {
	v := reflect.ValueOf(val)
	for {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return true
			}
			v = v.Elem()
		} else {
			break
		}
	}
	return v.IsZero()
}
