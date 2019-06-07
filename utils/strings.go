package utils

func InSliceOfStrings(s string, sl []string) bool {
	for _, item := range sl {
		if item == s {
			return true
		}
	}

	return false
}

func InSliceOfUint64(u uint64, sl []uint64) bool {
	for _, item := range sl {
		if item == u {
			return true
		}
	}

	return false
}
