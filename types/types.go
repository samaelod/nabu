package types

type Config struct {
	Globals        Globals
	Endpoints      []Endpoint
	Messages       []Message
	MessagesByFrom map[int][]Message // Pre-indexed messages by sender endpoint ID
}

// IndexMessages populates MessagesByFrom for O(1) lookup by sender
func (c *Config) IndexMessages() {
	c.MessagesByFrom = make(map[int][]Message, len(c.Endpoints))
	for _, m := range c.Messages {
		c.MessagesByFrom[m.From] = append(c.MessagesByFrom[m.From], m)
	}
}

type Globals struct {
	Protocol string
	PlayMode string
	Timeout  int // ms
	Delay    int // ms
	LogLines int // Max lines in memory buffer (default 1000)
}

type Endpoint struct {
	ID      int
	Kind    string // "server" | "client"
	Address string
	Port    int
}

type Message struct {
	From   int
	To     int
	Kind   string // "syn", "ack", "data", etc.
	Value  string // hex
	TDelta int    // ms since previous message
}

type EndpointStatus int

const (
	StatusIdle EndpointStatus = iota
	StatusRunning
	StatusCompleted
	StatusError
)
