package event

func (e *Event) mouseClickDown() {
	e.lock.Lock()
	events, ok := e.MouseClick[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for _, event := range events {
		event.Down(&EVENT_MouseButton{
			X:      int(e.getMouseButtonEvent().x),
			Y:      int(e.getMouseButtonEvent().y),
			Button: int(e.getMouseButtonEvent().button),
		})
	}
}

func (e *Event) mouseClickUp() {
	e.lock.Lock()
	events, ok := e.MouseClick[e.TopScreenName]
	e.lock.Unlock()
	if !ok {
		return
	}
	for _, event := range events {
		event.Up(&EVENT_MouseButton{
			X:      int(e.getMouseButtonEvent().x),
			Y:      int(e.getMouseButtonEvent().y),
			Button: int(e.getMouseButtonEvent().button),
		})
	}
}

func (e *Event) AddMouseClickEvent(screenName string, mce MouseClickEvent) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.MouseClick[screenName] = append(e.MouseClick[screenName], mce)
}
