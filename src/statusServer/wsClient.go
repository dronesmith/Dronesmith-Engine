package statusServer

import (
  "fmt"
  "io"

  "config"
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
  config.Log(config.LOG_INFO, "ss:  Killing Client")
  c.quit <-true
}

func (c *Client) Listener() {
  go c.txListener()
  c.rxListener()
}

func (c *Client) rxListener() {
  for {
    select {
    case <- c.quit:
      c.server.RmClient(c)
      c.quit <-true // kill tx
      return

    default:
      var buf string
      if err := websocket.JSON.Receive(c.ws, &buf); err == io.EOF {
				c.quit <- true
			} else if err != nil {
        c.quit <- true
				panic(err)
      } // ignore reads that aren't EOF
    }
  }
}

func (c *Client) txListener() {
  for {
    select {
    case data := <-c.send: // got data to send from write method.
      go func() {
        if err := websocket.JSON.Send(c.ws, data); err != nil {
          config.Log(config.LOG_ERROR, err, data.Altitude)
        }
      }()

    case <-c.quit: // kill request.
      c.server.RmClient(c)
      c.quit <-true // kill rx
      return
    }
  }
}
