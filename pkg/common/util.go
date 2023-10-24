package common

func StringPointer(s string) *string {
	return &s
}

func BoolPointer(b bool) *bool {
	return &b
}

func IntPointer(i int) *int {
	return &i
}

func Float64Pointer(f float64) *float64 {
	return &f
}
