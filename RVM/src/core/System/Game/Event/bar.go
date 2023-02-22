package event

func (e *Event) barDown() {
	e.lock.Lock()
	events, ok := e.Bar[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for n, event := range events {
		if event.Down(&EVENT_MouseButton{
			X:      int(e.getMouseButtonEvent().x),
			Y:      int(e.getMouseButtonEvent().y),
			Button: int(e.getMouseButtonEvent().button),
		}) {
			events[n].IsNowDown = true
		}
	}
}

func (e *Event) barUp() {
	e.lock.Lock()
	events, ok := e.Bar[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for n, event := range events {
		if event.IsNowDown {
			event.Up(&EVENT_MouseButton{
				X:      int(e.getMouseButtonEvent().x),
				Y:      int(e.getMouseButtonEvent().y),
				Button: int(e.getMouseButtonEvent().button),
			})
			events[n].IsNowDown = false
		}
	}
}

func (e *Event) barScroll() {
	e.lock.Lock()
	events, ok := e.Bar[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for _, event := range events {
		if event.IsNowDown {
			event.Scroll(&EVENT_MouseMotion{
				X:    int(e.getMouseMotionEvent().x),
				Y:    int(e.getMouseMotionEvent().y),
				Xrel: int(e.getMouseMotionEvent().xrel),
				Yrel: int(e.getMouseMotionEvent().yrel),
			})
		}
	}
}

/* -- Util -- */

func (e *Event) AddBarEvent(screenName string, be BarEvent) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.Bar[screenName] = append(e.Bar[screenName], be)
}
