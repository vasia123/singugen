package comms

import (
	"sync"
	"testing"
	"time"
)

func TestBus_SubscribeAndSend(t *testing.T) {
	b := New()
	ch := b.Subscribe("agent1")

	err := b.Send(Message{From: "agent2", To: "agent1", Content: "hello"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	select {
	case msg := <-ch:
		if msg.Content != "hello" || msg.From != "agent2" {
			t.Errorf("msg = %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("no message received")
	}
}

func TestBus_SendToUnknown(t *testing.T) {
	b := New()

	err := b.Send(Message{From: "a", To: "nonexistent", Content: "x"})
	if err == nil {
		t.Error("send to unknown should return error")
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	b := New()
	b.Subscribe("agent1")
	b.Unsubscribe("agent1")

	err := b.Send(Message{From: "a", To: "agent1", Content: "x"})
	if err == nil {
		t.Error("send after unsubscribe should return error")
	}
}

func TestBus_Broadcast(t *testing.T) {
	b := New()
	ch1 := b.Subscribe("agent1")
	ch2 := b.Subscribe("agent2")
	b.Subscribe("sender")

	b.Broadcast("sender", "announcement")

	// agent1 and agent2 should receive, sender should not.
	for _, ch := range []<-chan Message{ch1, ch2} {
		select {
		case msg := <-ch:
			if msg.Content != "announcement" {
				t.Errorf("content = %q", msg.Content)
			}
		case <-time.After(time.Second):
			t.Fatal("no broadcast received")
		}
	}
}

func TestBus_ConcurrentAccess(t *testing.T) {
	b := New()
	var wg sync.WaitGroup

	for i := range 10 {
		name := "agent" + string(rune('0'+i))
		b.Subscribe(name)
	}

	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			from := "agent" + string(rune('0'+n))
			to := "agent" + string(rune('0'+(n+1)%10))
			b.Send(Message{From: from, To: to, Content: "msg"})
		}(i)
	}

	wg.Wait()
}
