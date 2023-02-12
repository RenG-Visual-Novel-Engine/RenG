package game

var (
	DefinedTextSpeed = 20
)

// Private
func (g *Game) setNowName(n string) {
	*g.nowName = n
}

// Private
func (g *Game) setNowText(t string) {
	*g.nowText = t
}
