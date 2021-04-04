package fcm

import (
	"github.com/raf924/connector-api/pkg/gen"
	"sync"
)

type MessageQueue interface {
	Push(packet *gen.BotPacket)
	Pop() *gen.BotPacket
}

func NewMessageQueue() MessageQueue {
	lock := &sync.Mutex{}
	return &messageQueue{
		cond:     sync.NewCond(lock),
		messages: nil,
	}
}

type messageQueue struct {
	cond     *sync.Cond
	messages []*gen.BotPacket
}

func (q *messageQueue) Push(packet *gen.BotPacket) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.messages = append(q.messages, packet)
	q.cond.Signal()
}

func (q *messageQueue) Pop() *gen.BotPacket {
	println("Locking")
	q.cond.L.Lock()
	println("Locked")
	defer q.cond.L.Unlock()
	if len(q.messages) == 0 {
		println("Waiting")
		q.cond.Wait()
	}
	m := q.messages[0]
	q.messages[0] = nil
	println("Popped", m.Message)
	q.messages = q.messages[1:]
	return m
}
