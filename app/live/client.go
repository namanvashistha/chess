package live

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"chess-engine/app/store"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second // under pongWait, so a live peer always answers in time
	maxMessage = 1024             // bytes; a move message is about 40
	sendBuffer = 64               // messages queued per client before it counts as stuck
)

// Client is one WebSocket connection to one game. Its writes go through its
// own goroutine, so a slow client delays nobody else; if its queue fills it is
// disconnected.
type Client struct {
	conn *websocket.Conn
	user *store.User // nil for a spectator who is not signed in
	send chan []byte

	closeOnce sync.Once
	closed    chan struct{}
}

// queue sends msg to the client without blocking. It is called only from the
// room goroutine.
func (c *Client) queue(msg any) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
		c.close()
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.conn.Close()
	})
}

// Serve runs a WebSocket connection for a game until it closes. user is nil for
// a spectator who is not signed in.
func (m *Manager) Serve(ctx context.Context, gameID string, conn *websocket.Conn, user *store.User) error {
	c := &Client{conn: conn, user: user, send: make(chan []byte, sendBuffer), closed: make(chan struct{})}
	defer c.close()

	var room *Room
	if err := m.do(ctx, gameID, func(r *Room) { room = r; r.attach(c) }); err != nil {
		return err
	}
	defer func() {
		select {
		case room.inbox <- func() { room.detach(c) }:
		case <-room.done:
		}
	}()

	go c.writeLoop()
	c.readLoop(room)
	return nil
}

func (c *Client) readLoop(room *Room) {
	c.conn.SetReadLimit(maxMessage)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var in inbound
		bad := json.Unmarshal(data, &in) != nil
		f := func() {
			if bad {
				room.fail(c, CodeBadMessage, "messages must be JSON objects")
				return
			}
			room.handle(c, in)
		}
		select {
		case room.inbox <- f:
		case <-room.done:
			return
		case <-c.closed:
			return
		}
	}
}

func (c *Client) writeLoop() {
	ping := time.NewTicker(pingPeriod)
	defer ping.Stop()
	defer c.close()
	for {
		select {
		case b := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		case <-ping.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closed:
			return
		}
	}
}
