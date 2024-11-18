package ptr

func ValueToPointer[V any](v V) *V {
	return &v
}

func PointerToValue[V any](v *V) V {
	if v == nil {
		return *new(V)
	}
	return *v
}
