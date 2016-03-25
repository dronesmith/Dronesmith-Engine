package statusServer

import (
  "bytes"
  "fmt"
  "html/template"
  "io"
  "encoding/json"
  "log"
  "net/http"
  "path/filepath"
  // "time"

  "fmulink"
  "golang.org/x/net/websocket"
)

const (
  STATIC_PATH = "assets/public"
  TMPL_PATH = "assets/templates"
  SERVE_ADDRESS = ":8080"
  SOCKET_ADDRESS = "ws://localhost:8080/api/fmu"

  LUCI_SETUP_TITLE = "Luci: First Time Setup"
)

type StatusServer struct {
  address       string
  wsClients     map[ClientId]*Client
  fileServer    http.ServeMux

  // events
  addClient     chan *Client
  delClient     chan *Client
  fmuEvent      chan *fmulink.Fmu
  quit          chan bool
  err           chan error

  // index Template
  indexTmpl     []byte
}

func NewStatusServer(address string) (*StatusServer) {
  return &StatusServer {
    address,
    make(map[ClientId]*Client),
    *http.NewServeMux(),
    make(chan *Client),
    make(chan *Client),
    make(chan *fmulink.Fmu),
    make(chan bool),
    make(chan error),
    nil,
  }
}

// =============================================================================
// Status Server
// =============================================================================

func (s *StatusServer) Serve() {
  // Set up routing table
  s.fileServer.Handle(  "/",            http.FileServer(http.Dir(STATIC_PATH)))
  http.HandleFunc(    "/",            s.rootHandler)
  http.HandleFunc(    "/api/setup",   s.setupResponse)
  http.HandleFunc(    "/api/aps",     s.apsResponse)
  http.Handle(        "/api/fmu",     websocket.Handler(s.wsOnConnect))

  // Compile templates
  if err := s.initTemplates(TMPL_PATH); err != nil {
		log.Fatal(err)
	} else {
    go s.wsListener()
    log.Fatal(http.ListenAndServe(s.address, nil))
  }
}

func (s *StatusServer) initTemplates(root string) error {
  index := filepath.Join(root, "index.tmpl")
	ui, err := template.ParseFiles(index)
	if err != nil {
		return fmt.Errorf("parse index.tmpl: %v", err)
	}

  buffer := new(bytes.Buffer)

  templateData := struct {
    Title string
    SocketAddress string
  }{LUCI_SETUP_TITLE, SOCKET_ADDRESS}

  if err := ui.Execute(buffer, templateData); err != nil {
		return fmt.Errorf("render UI: %v", err)
	}

	s.indexTmpl = buffer.Bytes()
  return nil
}

func (s *StatusServer) rootHandler(w http.ResponseWriter, r* http.Request) {
  // Default error handler for bad web reqeusts. Sends out a 500.
  defer s.handler500(&w)

  // If it's root path, render index
  if r.URL.Path == "/" {
	  if err := s.renderIndex(w); err != nil {
		  panic(err)
	  }
    return
  }

  // Else, serve static content
  s.fileServer.ServeHTTP(w, r)

  // Handle 404s
  // if r.URL.Path != "/" {
  //   http.Error(w, http.StatusText(404), 404)
  //   return
  // }
}

func (s *StatusServer) renderIndex(w io.Writer) error {
	if s.indexTmpl == nil {
		return fmt.Errorf("Could not load root template.")
	}
  _, err := w.Write(s.indexTmpl)
	return err
}

func (s *StatusServer) handler500(w* http.ResponseWriter) {
  if r := recover(); r != nil {
    http.Error(*w, http.StatusText(500) + ": " + r.(string), 500)
  }
}

// =============================================================================
// Websocket Handling
// =============================================================================

func (s *StatusServer) RmClient(c *Client) {
  s.delClient <-c
}

func (s *StatusServer) sendAll(data *fmulink.Fmu) {
  for _, c := range s.wsClients {
    if err := c.Send(data); err != nil {
      s.err <-err
    }
  }
}

func (s *StatusServer) wsListener() {
  for {
    select {
    case c := <-s.addClient: // new websocket connection
      s.wsClients[c.id] = c

    case c := <-s.delClient: // either a client disconnected, or request term
      delete(s.wsClients, c.id)

    case data := <-s.fmuEvent: // got status update
      s.sendAll(data)

    case err := <-s.err: // error
      log.Println("Websocket Error:", err.Error())

    case <-s.quit: // kill server
      return
    }
  }
}

func (s *StatusServer) wsOnConnect(ws *websocket.Conn) {
  // Deal with websocket Errors
  defer func() {
    recover()
    if err := ws.Close(); err != nil {
      s.err <- err
    }
  }()

  // Create a websocket client for the connection
  if client, err := NewClient(ws, s); err != nil {
    panic(err)
  } else {
    s.addClient <-client
    go client.Listener()
  }
}

// =============================================================================
// API: /api/setup
// =============================================================================

type APIPostSetupReq struct {
  Email string
  Password string
}

type APIPostSetupRes struct {
  Error string
}

func (s *StatusServer) setupResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "POST":
    r.ParseForm()
    var obj APIPostSetupReq
    for key, _ := range r.Form {
      if err := json.Unmarshal([]byte(key), &obj); err != nil {
        panic(err.Error())
      } else {
        // TODO
        fmt.Println("TODO: Auth with DSC")
        res := APIPostSetupRes{Error: "Not implemented yet."}
        if data, err := json.Marshal(res); err != nil {
          panic(err)
        } else {
          if _ , err := w.Write(data); err != nil {
            panic(err)
          }
        }
      }
    }
  default:
    http.Error(w, http.StatusText(404), 404)
  }
}

// =============================================================================
// API: /api/aps
// =============================================================================


type NetworkType struct {
  SSID            string
  Kind            string
  NeedsPassword   bool
}

type APIGetApsRes struct {
  Error     string
  Networks []NetworkType
}

type APIPostApsRes struct {
  Error     string
  Status    string
}

func (s *StatusServer) apsResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "POST":
    // TODO
    fmt.Println("TODO: POST Setup wireless network")
    fallthrough
  case "GET":
    // TODO
    fmt.Println("TODO: GET Setup wireless network")
    fallthrough
  default:
    http.Error(w, http.StatusText(404), 404)
  }
}
