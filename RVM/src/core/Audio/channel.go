package audio

type Channel struct {
	Playing bool
}

func NewChannel() *Channel {
	return &Channel{Playing: false}
}
