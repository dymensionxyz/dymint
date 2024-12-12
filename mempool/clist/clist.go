package clist

import (
	"fmt"
	"sync"

	tmsync "github.com/tendermint/tendermint/libs/sync"
)

const MaxLength = int(^uint(0) >> 1)

type CElement struct {
	mtx        tmsync.RWMutex
	prev       *CElement
	prevWg     *sync.WaitGroup
	prevWaitCh chan struct{}
	next       *CElement
	nextWg     *sync.WaitGroup
	nextWaitCh chan struct{}
	removed    bool

	Value interface{}
}

func (e *CElement) NextWait() *CElement {
	for {
		e.mtx.RLock()
		next := e.next
		nextWg := e.nextWg
		removed := e.removed
		e.mtx.RUnlock()

		if next != nil || removed {
			return next
		}

		nextWg.Wait()

	}
}

func (e *CElement) PrevWait() *CElement {
	for {
		e.mtx.RLock()
		prev := e.prev
		prevWg := e.prevWg
		removed := e.removed
		e.mtx.RUnlock()

		if prev != nil || removed {
			return prev
		}

		prevWg.Wait()
	}
}

func (e *CElement) PrevWaitChan() <-chan struct{} {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	return e.prevWaitCh
}

func (e *CElement) NextWaitChan() <-chan struct{} {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	return e.nextWaitCh
}

func (e *CElement) Next() *CElement {
	e.mtx.RLock()
	val := e.next
	e.mtx.RUnlock()
	return val
}

func (e *CElement) Prev() *CElement {
	e.mtx.RLock()
	prev := e.prev
	e.mtx.RUnlock()
	return prev
}

func (e *CElement) Removed() bool {
	e.mtx.RLock()
	isRemoved := e.removed
	e.mtx.RUnlock()
	return isRemoved
}

func (e *CElement) DetachNext() {
	e.mtx.Lock()
	if !e.removed {
		e.mtx.Unlock()
		panic("DetachNext() must be called after Remove(e)")
	}
	e.next = nil
	e.mtx.Unlock()
}

func (e *CElement) DetachPrev() {
	e.mtx.Lock()
	if !e.removed {
		e.mtx.Unlock()
		panic("DetachPrev() must be called after Remove(e)")
	}
	e.prev = nil
	e.mtx.Unlock()
}

func (e *CElement) SetNext(newNext *CElement) {
	e.mtx.Lock()

	oldNext := e.next
	e.next = newNext
	if oldNext != nil && newNext == nil {

		e.nextWg = waitGroup1()
		e.nextWaitCh = make(chan struct{})
	}
	if oldNext == nil && newNext != nil {
		e.nextWg.Done()
		close(e.nextWaitCh)
	}
	e.mtx.Unlock()
}

func (e *CElement) SetPrev(newPrev *CElement) {
	e.mtx.Lock()

	oldPrev := e.prev
	e.prev = newPrev
	if oldPrev != nil && newPrev == nil {
		e.prevWg = waitGroup1()
		e.prevWaitCh = make(chan struct{})
	}
	if oldPrev == nil && newPrev != nil {
		e.prevWg.Done()
		close(e.prevWaitCh)
	}
	e.mtx.Unlock()
}

func (e *CElement) SetRemoved() {
	e.mtx.Lock()

	e.removed = true

	if e.prev == nil {
		e.prevWg.Done()
		close(e.prevWaitCh)
	}
	if e.next == nil {
		e.nextWg.Done()
		close(e.nextWaitCh)
	}
	e.mtx.Unlock()
}

type CList struct {
	mtx    tmsync.RWMutex
	wg     *sync.WaitGroup
	waitCh chan struct{}
	head   *CElement
	tail   *CElement
	len    int
	maxLen int
}

func (l *CList) Init() *CList {
	l.mtx.Lock()

	l.wg = waitGroup1()
	l.waitCh = make(chan struct{})
	l.head = nil
	l.tail = nil
	l.len = 0
	l.mtx.Unlock()
	return l
}

func New() *CList { return newWithMax(MaxLength) }

func newWithMax(maxLength int) *CList {
	l := new(CList)
	l.maxLen = maxLength
	return l.Init()
}

func (l *CList) Len() int {
	l.mtx.RLock()
	len := l.len
	l.mtx.RUnlock()
	return len
}

func (l *CList) Front() *CElement {
	l.mtx.RLock()
	head := l.head
	l.mtx.RUnlock()
	return head
}

func (l *CList) FrontWait() *CElement {
	for {
		l.mtx.RLock()
		head := l.head
		wg := l.wg
		l.mtx.RUnlock()

		if head != nil {
			return head
		}
		wg.Wait()

	}
}

func (l *CList) Back() *CElement {
	l.mtx.RLock()
	back := l.tail
	l.mtx.RUnlock()
	return back
}

func (l *CList) BackWait() *CElement {
	for {
		l.mtx.RLock()
		tail := l.tail
		wg := l.wg
		l.mtx.RUnlock()

		if tail != nil {
			return tail
		}
		wg.Wait()

	}
}

func (l *CList) WaitChan() <-chan struct{} {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	return l.waitCh
}

func (l *CList) PushBack(v interface{}) *CElement {
	l.mtx.Lock()

	e := &CElement{
		prev:       nil,
		prevWg:     waitGroup1(),
		prevWaitCh: make(chan struct{}),
		next:       nil,
		nextWg:     waitGroup1(),
		nextWaitCh: make(chan struct{}),
		removed:    false,
		Value:      v,
	}

	if l.len == 0 {
		l.wg.Done()
		close(l.waitCh)
	}
	if l.len >= l.maxLen {
		panic(fmt.Sprintf("clist: maximum length list reached %d", l.maxLen))
	}
	l.len++

	if l.tail == nil {
		l.head = e
		l.tail = e
	} else {
		e.SetPrev(l.tail)
		l.tail.SetNext(e)
		l.tail = e
	}
	l.mtx.Unlock()
	return e
}

func (l *CList) Remove(e *CElement) interface{} {
	l.mtx.Lock()

	prev := e.Prev()
	next := e.Next()

	if l.head == nil || l.tail == nil {
		l.mtx.Unlock()
		panic("Remove(e) on empty CList")
	}
	if prev == nil && l.head != e {
		l.mtx.Unlock()
		panic("Remove(e) with false head")
	}
	if next == nil && l.tail != e {
		l.mtx.Unlock()
		panic("Remove(e) with false tail")
	}

	if l.len == 1 {
		l.wg = waitGroup1()
		l.waitCh = make(chan struct{})
	}

	l.len--

	if prev == nil {
		l.head = next
	} else {
		prev.SetNext(next)
	}
	if next == nil {
		l.tail = prev
	} else {
		next.SetPrev(prev)
	}

	e.SetRemoved()

	l.mtx.Unlock()
	return e.Value
}

func waitGroup1() (wg *sync.WaitGroup) {
	wg = &sync.WaitGroup{}
	wg.Add(1)
	return
}
