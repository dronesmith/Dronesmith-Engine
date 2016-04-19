package statusServer

import (
  "bytes"
  "fmt"
  "html/template"
  "io"
  "os"
  "encoding/json"
  "log"
  "net/http"
  "path/filepath"
  "time"

  "fmulink"
  "cloudlink"
  "golang.org/x/net/websocket"
  "config"
)

const (
  STATIC_PATH = "assets/public"
  TMPL_PATH = "assets/templates"
  // SERVE_ADDRESS = ":8080"

  LUCI_SETUP_TITLE = "Luci: First Time Setup"
  LUCI_MAIN_TITLE = "Luci: Status"
)

var (
  SOCKET_ADDRESS = "ws://" + *config.StatusAddress + "/api/fmu"
  NETWORKS_FILE = *config.SetupPath + "networks.txt"
)

type StatusServer struct {
  address       string
  wsClients     map[ClientId]*Client
  fileServer    http.ServeMux
  cloud         *cloudlink.CloudLink

  // events
  addClient     chan *Client
  delClient     chan *Client
  fmuEvent      chan *fmulink.Fmu
  quit          chan bool
  err           chan error

  // index Template
  indexTmpl     []byte
}

func NewStatusServer(address string, cloud *cloudlink.CloudLink) (*StatusServer) {
  return &StatusServer {
    address,
    make(map[ClientId]*Client),
    *http.NewServeMux(),
    cloud,
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
  s.fileServer.Handle("/",            http.FileServer(http.Dir(STATIC_PATH)))
  http.HandleFunc(    "/",            s.rootHandler)
  http.HandleFunc(    "/api/output",  s.outResponse)
  http.HandleFunc(    "/api/setup",   s.setupResponse)
  http.HandleFunc(    "/api/aps",     s.apsResponse)
  http.Handle(        "/api/fmu",     websocket.Handler(s.wsOnConnect))

  // Compile templates
  if err := s.initTemplates(TMPL_PATH); err != nil {
    config.Log(config.LOG_ERROR, "ss: ", err)
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

  var route, ctrl, title string

  buffer := new(bytes.Buffer)

  e, p := s.cloud.GetStore().Get()

  if e == "" || p == "" {
    route = "main.html"
    ctrl = "MainCtrl"
    title = LUCI_SETUP_TITLE
  } else {
    route = "status.html"
    ctrl = "StatusCtrl"
    title = LUCI_MAIN_TITLE
  }

  templateData := struct {
    Title string
    SocketAddress string
    SelectedRoute string
    SelectedCtrl string
    Version string
  }{title, SOCKET_ADDRESS, route, ctrl, config.Version}

  if err := ui.Execute(buffer, templateData); err != nil {
		return fmt.Errorf("render UI: %v", err)
	}

	s.indexTmpl = buffer.Bytes()
  return nil
}

func (s *StatusServer) rootHandler(w http.ResponseWriter, r* http.Request) {
  // Default error handler for bad web reqeusts. Sends out a 500.
  defer s.handler500(&w)

  // Set CORS
  w.Header().Set("Access-Control-Allow-Origin", "*")

  // If it's root path, render index
  if r.URL.Path == "/" || r.URL.Path == "/status" {
    config.Log(config.LOG_INFO, "ss: ", "[GET] Connect request")
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

func (s *StatusServer) periodicFmuStatus(d time.Duration) {
  for ticker := time.NewTicker(d); ; {
    select {
    case <-ticker.C:
      s.fmuEvent <- fmulink.GetData()

    case <-s.quit:
      s.quit <-true // kill wsListener
      return
    }
  }
}

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
  go s.periodicFmuStatus(time.Second) // start getting updates from fmulink

  for {
    select {
    case c := <-s.addClient: // new websocket connection
      s.wsClients[c.id] = c
      config.Log(config.LOG_INFO, "ss: ", "SOCKET ADD | Total connections:", len(s.wsClients))

    case c := <-s.delClient: // either a client disconnected, or request term
      delete(s.wsClients, c.id)
      config.Log(config.LOG_INFO, "ss: ", "SOCKET DEL | Total connections:", len(s.wsClients))

    case data := <-s.fmuEvent: // got status update
      s.sendAll(data)

    case err := <-s.err: // error
      config.Log(config.LOG_ERROR, "ss: ", "Websocket Error:", err.Error())

    case <-s.quit: // kill server
      s.quit <- true // kill periodicFmuStatus
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
    client.Listener() // making this async will kill the socket connection
  }
}

// =============================================================================
// API: /api/output [POST]
// =============================================================================

type APIPostOutputReq struct {
  Address string
}

type APIPostOutputRes struct {
  Error string
  Status string
}

func (s *StatusServer) outResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "POST":
    var obj APIPostOutputReq
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&obj)
    if err != nil {
      panic(err)
    }

    var res APIPostOutputRes
    config.Log(config.LOG_INFO, "ss: ", "Adding output address:", obj.Address)
    err = fmulink.Outputs.Add(obj.Address)
    if err != nil {
      config.Log(config.LOG_ERROR, "ss: ", err.Error())
      res = APIPostOutputRes{Error: err.Error(), Status: "error"}
    } else {
      res = APIPostOutputRes{Error: "", Status: "OK"}
    }

    if data, err := json.Marshal(res); err != nil {
      panic(err)
    } else {
      if _ , err := w.Write(data); err != nil {
        panic(err)
      }
    }
  default:
    http.Error(w, http.StatusText(404), 404)
  }
}

// =============================================================================
// API: /api/setup [POST]
// =============================================================================

type APIPostSetupReq struct {
  Email string        `json:"email"`
  Password string     `json:"password"`
}

type APIPostSetupRes struct {
  Status      string  `json:"status"`
  Error       string  `json:"error"`
}

type APIGetSetUpRes struct {
  Step        int     `json:"step"`
  Error       string  `json:"error"`
}

func (s *StatusServer) setupResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "GET":
    var obj APIGetSetUpRes

    if ip, _, err := checkIP(); err != nil {
      obj.Error = err.Error()
    } else if ip { // connected. Load next step.
      obj.Step = 2
    } else { // not connected.
      obj.Step = 1
    }

    if data, err := json.Marshal(obj); err != nil {
      panic(err)
    } else {
      if _ , err := w.Write(data); err != nil {
        panic(err)
      }
    }

  case "POST":
    var obj APIPostSetupReq
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&obj)
    if err != nil {
      panic(err)
    }

    store := s.cloud.GetStore()
    e, p := store.Get()

    if e != "" || p != "" {
      err = fmt.Errorf("Already activated.")
    } else {
      err = store.Set(obj.Email, obj.Password)
    }

    var res APIPostSetupRes
    if err != nil {
      config.Log(config.LOG_ERROR, "ss: ", err.Error())
      res = APIPostSetupRes{Error: err.Error(), Status: "error"}
      if data, err := json.Marshal(res); err != nil {
        panic(err)
      } else {
        if _ , err := w.Write(data); err != nil {
          panic(err)
        }
      }
    } else {

      // pend for auth
      // This blocks for a few seconds.
      auth := s.cloud.IsOnline()

      if !auth {
        store.Del()
        res = APIPostSetupRes{Error: "Authentication failed.", Status: "error"}
      } else {
        res = APIPostSetupRes{Error: "", Status: "OK"}
        s.initTemplates(TMPL_PATH)
      }

      if data, err := json.Marshal(res); err != nil {
        panic(err)
      } else {
        if _ , err := w.Write(data); err != nil {
          panic(err)
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

type APIGetApsRes struct {
  Error     string            `json:"error"`
  Networks  map[string]string `json:"aps"`
}

type APIPostApsReq struct {
  SysName   string  `json:"name,omitempty"`
  Ssid      string  `json:"ssid"`
  Protocol  string  `json:"protocol"`
  Password  string  `json:"password,omitempty"`
  Username  string  `json:"username,omitempty"`
}

type APIPostApsRes struct {
  Ssid      string  `json:"ssid,omitempty"`
  Name      string  `json:"name,omitempty"`
  Ip        string  `json:"ip,omitempty"`
  Error     string  `json:"error"`
}

func (s *StatusServer) apsResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "POST":
    var obj APIPostApsReq
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&obj)
    var name string
    if err != nil {
      panic(err)
    }

    // change to "configure_edison"
    if obj.SysName != "" {
      name = obj.SysName
    } else {
      name = "luci"
    }

    err = setName(name)

    // get name
    var namesMap map[string]string
    namesMap, err = getNames()

    switch obj.Protocol {
    case "OPEN":

      _, err = runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid)
    case "WEP":
      if (len(obj.Password) == 5 || len(obj.Password) == 13) {
        _, err = runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid, obj.Password)
      } else {
        err = fmt.Errorf("Network password must be either 5 or 13 characters in length.")
      }
    case "WPA-PSK":
      if (len(obj.Password) >= 8 && len(obj.Password) <= 63) {
        _, err = runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid, obj.Password)
      } else {
        err = fmt.Errorf("Network password must be between 8 and 63 characters in length.")
      }
    case "WPA_EAP":
      _, err = runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid, obj.Username, obj.Password)
    default:
      err = fmt.Errorf("Invalid or unsupported network protocol.")
    }

    // wait to validate wifi
    time.Sleep(5 * time.Second)

    var ipAddr string
    _, ipAddr, err = checkIP()

    var res *APIPostApsRes

    if err != nil {
      res = &APIPostApsRes{Error: err.Error(),}
    } else {
      res = &APIPostApsRes{obj.Ssid, namesMap["ssid"], ipAddr, "",}
    }


    if data, err := json.Marshal(res); err != nil {
      panic(err)
    } else {
      if _ , err := w.Write(data); err != nil {
        panic(err)
      }
    }

  case "GET":
    var res APIGetApsRes
    // grab network file
    if file, err := os.Open(NETWORKS_FILE); err != nil {
      res.Error = err.Error()
    } else {
      rbuff := make([]byte, 1024)
      // aps := make(map[string]string)
      if cnt, err := file.Read(rbuff); err != nil {
        res.Error = err.Error()
      } else {
        if err := json.Unmarshal(rbuff[:cnt], &res.Networks); err != nil {
          res.Error = err.Error()
        }
      }
    }

    if data, err := json.Marshal(res); err != nil {
      panic(err)
    } else {
      if _ , err := w.Write(data); err != nil {
        panic(err)
      }
    }
  default:
    http.Error(w, http.StatusText(404), 404)
  }
}
