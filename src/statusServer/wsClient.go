package statusServer

import (
  "fmt"

  "fmulink"
  "golang.org/x/net/websocket"
)

type ClientId uint

var (
  ids ClientId = 0
)

type Client struct {
  id      ClientId
  ws      *websocket.Conn
  server  *StatusServer
  send    chan *fmulink.Fmu
  quit    chan bool
}

func NewClient(ws *websocket.Conn, s *StatusServer) (*Client, error) {
  if ws == nil || s == nil {
    return nil,
      fmt.Errorf("Cannot construct a client with nil params.\nWebsocket: %v\nServer: %v\n", ws, s)
  }

  ids++

  return &Client{
    ids,
    ws,
    s,
    make(chan *fmulink.Fmu),
    make(chan bool),
  }, nil
}

// NOTE blocking.
func (c *Client) Send(data *fmulink.Fmu) error {
	select {
	case c.send <-data: // Signal the listener to fire off

	default: // send failed. Assume client disconnected
    c.server.RmClient(c)
		err := fmt.Errorf("client %d is disconnected.", c.id)
    return err
	}

  return nil
}

func (c *Client) Ws() *websocket.Conn {
  return c.ws
}

func (c *Client) Server() *StatusServer {
  return c.server
}

func (c *Client) Kill() {
  c.quit <-true
}

// NOTE blocking. This should be run async.
func (c *Client) Listener() {
  for {
    select {
    case data := <-c.send: // got data to send from write method.
      fmt.Println("Send:", data)
      websocket.JSON.Send(c.ws, data)

    case <-c.quit: // kill request.
      c.server.RmClient(c)
      return
    }
  }
}
