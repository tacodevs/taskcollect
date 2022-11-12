package main

//func contains(strSlice []string, str string) bool {
//	for i := 0; i < len(strSlice); i++ {
//		if strSlice[i] == str {
//			return true
//		}
//	}
//	return false
//}

func contains[T comparable](arr []T, elem T) bool {
	for _, val := range arr {
		if val == elem {
			//fmt.Printf("%v - %v\n", val, elem)
			return true
		}
	}
	return false
}

// Remove an item from a slice
func remove[T comparable](arr []T, item T) []T {
	for i, other := range arr {
		if other == item {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}
