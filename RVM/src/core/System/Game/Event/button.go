package event

func (e *Event) buttonUp() {
	e.lock.Lock()
	events, ok := e.Button[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for _, event := range events {
		event.Action(&EVENT_MouseButton{
			X:      int(e.getMouseButtonEvent().x),
			Y:      int(e.getMouseButtonEvent().y),
			Button: int(e.getMouseButtonEvent().button),
		})
	}
}

func (e *Event) buttonHover() {
	e.lock.Lock()
	events, ok := e.Button[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for _, event := range events {
		event.Hover(&EVENT_MouseMotion{
			X: int(e.getMouseMotionEvent().x),
			Y: int(e.getMouseMotionEvent().y),
		})
	}

}

/* -- Util -- */

func (e *Event) AddButtonEvent(screenName string, be ButtonEvent) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.Button[screenName] = append(e.Button[screenName], be)
}
