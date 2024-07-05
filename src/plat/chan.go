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

func Send[T any](c chan T, v T) {
	c <- v
}

func Wait[T any](c chan T, done chan bool, jobs int) {
	for i := 0; i < jobs; i++ {
		<-done
	}
	close(done)
	close(c)
}
