package plat

type Pair[T, U any] struct {
	First  T
	Second U
}

func Mark[T any](done *int, c chan T) {
	*done++
	if *done == 0 {
		close(c)
	}
}
