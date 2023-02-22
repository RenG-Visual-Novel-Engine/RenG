package game

func (g *Game) StartLabel(name string, index int) {

	// label을 screen처럼 등록
	bps, ok := g.screenBps[name]
	if !ok {
		g.screenBps[name] = g.Graphic.GetCurrentTopRenderBps() + 1
		bps = g.screenBps[name]
		g.Graphic.AddScreenRenderBuffer()
	}

	//config 정보 삽입
	g.NowlabelName = name
	g.Event.TopScreenName = name

	//loadData를 위한 index
	g.labels[name].Obj = g.labels[name].Obj[index:]

	g.labelCallStack = append(g.labelCallStack, name)
	jumpLabel := g.labelEval(g.labels[name], name, bps, index)

	for jumpLabel != "" {
		g.NowlabelName = jumpLabel
		g.NowlabelIndex = 0
		g.Event.TopScreenName = jumpLabel
		g.labelCallStack = nil
		g.labelCallStack = append(g.labelCallStack, jumpLabel)
		jumpLabel = g.labelEval(g.labels[jumpLabel], jumpLabel, bps, index)
	}

	g.InActiveScreen(name)
}
