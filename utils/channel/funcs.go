package channel

func DrainForever[T any](chs ...<-chan T) {
	for _, ch := range chs {
		go func() {
			for {
				<-ch
			}
		}()
	}
}
