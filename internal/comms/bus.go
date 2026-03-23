package comms

import (
	"fmt"
	"sync"
)

const inboxSize = 64

// Message is sent between agents via the Bus.
type Message struct {
	From    string
	To      string
	Content string
}

// Bus provides channel-based inter-agent communication.
type Bus struct {
	mu   sync.RWMutex
	subs map[string]chan Message
}

// New creates a message bus.
func New() *Bus {
	return &Bus{subs: make(map[string]chan Message)}
}

// Subscribe creates an inbox channel for the named agent.
func (b *Bus) Subscribe(name string) <-chan Message {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Message, inboxSize)
	b.subs[name] = ch
	return ch
}

// Unsubscribe removes an agent's inbox.
func (b *Bus) Unsubscribe(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subs[name]; ok {
		close(ch)
		delete(b.subs, name)
	}
}

// Send delivers a message to a specific agent. Non-blocking.
func (b *Bus) Send(msg Message) error {
	b.mu.RLock()
	ch, ok := b.subs[msg.To]
	b.mu.RUnlock()

	if !ok {
		return fmt.Errorf("comms: agent %q not subscribed", msg.To)
	}

	select {
	case ch <- msg:
		return nil
	default:
		return fmt.Errorf("comms: inbox full for %q", msg.To)
	}
}

// Broadcast sends a message to all subscribed agents except the sender.
func (b *Bus) Broadcast(from, content string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for name, ch := range b.subs {
		if name == from {
			continue
		}
		select {
		case ch <- Message{From: from, To: name, Content: content}:
		default:
		}
	}
}
