package rlink

import (
	"fmt"
	"sync"
	"time"
)

const DefaultBufferSize int = 256
const DefaultDuration time.Duration = 3 * time.Second

// Unified message types that both UE and GNB use
type Message interface {
	GetType() string
}

// Shared connection structure used by both UE and GNB
type Connection struct {
	UEID       int64
	UEmsin     string
	GNBID      string
	UplinkCh   chan Message // UE → GNB
	DownlinkCh chan Message // GNB → UE
	Timeout    time.Duration
	closed     bool
	mu         sync.RWMutex
}

func NewConnection(
	ueID int64,
	msin string,
	gnbID string,
	bufferSize int,
	timeout time.Duration,
) *Connection {
	return &Connection{
		UEID:       ueID,
		UEmsin:     msin,
		GNBID:      gnbID,
		UplinkCh:   make(chan Message, bufferSize),
		DownlinkCh: make(chan Message, bufferSize),
		Timeout:    timeout,
	}
}

func (c *Connection) SendUplink(msg Message) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("connection closed")
	}

	select {
	case c.UplinkCh <- msg:
		//logger.RLinkConnStats[c.UEmsin].MessageSend.Add(1)
		return nil
	case <-time.After(c.Timeout):
		//logger.RLinkConnStats[c.UEmsin].MessageSendDropped.Add(1)
		return fmt.Errorf("timeout sending uplink message")
	}
}

func (c *Connection) SendDownlink(msg Message) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("connection closed")
	}

	select {
	case c.DownlinkCh <- msg:
		//logger.RLinkConnStats[c.GNBID].MessageSend.Add(1)
		return nil
	case <-time.After(c.Timeout):
		//logger.RLinkConnStats[c.GNBID].MessageSendDropped.Add(1)
		return fmt.Errorf("timeout sending downlink message")
	}
}

func (c *Connection) GetUplinkChan() <-chan Message {
	return c.UplinkCh
}

func (c *Connection) GetDownlinkChan() <-chan Message {
	return c.DownlinkCh
}

func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.UplinkCh)
		close(c.DownlinkCh)
	}
}

func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

func ConnectionKey(ueID int64, gnbID string) string {
	return fmt.Sprintf("%d:%s", ueID, gnbID)
}
