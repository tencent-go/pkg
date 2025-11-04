package util

func SliceConv[T any, V any](s []T, f func(T) V) []V {
	var result = make([]V, 0, len(s))
	for _, v := range s {
		result = append(result, f(v))
	}
	return result
}
