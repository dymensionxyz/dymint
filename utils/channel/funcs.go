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



type Nudger struct {
	C chan struct{} 
}

func NewNudger() *Nudger {
	return &Nudger{make(chan struct{})}
}


func (w Nudger) Nudge() {
	select {
	case w.C <- struct{}{}:
	default:
	}
}
