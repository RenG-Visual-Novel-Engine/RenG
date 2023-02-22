package game

func (g *Game) PlayMusic(path string, loop bool, ms int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Audio.PlayMusic(g.path+path, loop, ms)
}

func (g *Game) PlayChannel(chanName, path string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Audio.PlayChannel(chanName, g.path+path)
}

func (g *Game) GetMusicVolume() (volume int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.Audio.Music.GetVolume()
}

func (g *Game) GetChannelVolume(chanName string) (volume int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	return g.Audio.Channels[chanName].GetVolume()
}

func (g *Game) SetMusicVolume(volume int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Audio.Music.SetVolume(volume)
}

func (g *Game) SetChannelVolume(chanName string, volume int) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Audio.Channels[chanName].SetVolume(volume)
}

func (g *Game) CreateNewChannel(chanName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.Audio.MakeChan(chanName)
}
