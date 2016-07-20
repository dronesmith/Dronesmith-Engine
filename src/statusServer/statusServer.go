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
  // SERVE_ADDRESS = ":8080"

  LUCI_SETUP_TITLE = "Luci: First Time Setup"
  LUCI_MAIN_TITLE = "Luci: Status"
)

var (
  SOCKET_ADDRESS = "ws://" + *config.StatusAddress + "/api/fmu"
  NETWORKS_FILE = *config.SetupPath + "networks.txt"
  STATIC_PATH = *config.AssetsPath+"assets/public"
  TMPL_PATH = *config.AssetsPath+"assets/templates"
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
  http.HandleFunc(    "/api/logout",  s.logoutResponse)
  http.HandleFunc(    "/api/sensor",  s.sensorResponse)
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

  e := s.cloud.GetStore().Get("email")
  p := s.cloud.GetStore().Get("pass")

  for _, e := range s.cloud.GetStore().GetOutput() {
    fmulink.Outputs.Add(e)
  }

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
  if r.URL.Path == "/" || r.URL.Path == "/status" || r.URL.Path == "/wifi" {
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
      config.Log(config.LOG_INFO, "ss: ", "Kill periodic fmu status")
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
      if s.wsClients != nil {
        s.wsClients[c.id] = c
        config.Log(config.LOG_INFO, "ss: ", "SOCKET ADD | Total connections:", len(s.wsClients))
      } else {
        config.Log(config.LOG_WARN, "ss: ", "Attempted to add socket to a nil map!")
      }

    case c := <-s.delClient: // either a client disconnected, or request term
      delete(s.wsClients, c.id)
      config.Log(config.LOG_INFO, "ss: ", "SOCKET DEL | Total connections:", len(s.wsClients))

    case data := <-s.fmuEvent: // got status update
      s.sendAll(data)

    case err := <-s.err: // error
      config.Log(config.LOG_ERROR, "ss: ", "Websocket Error:", err.Error())

    case <-s.quit: // kill server
      config.Log(config.LOG_INFO, "ss: ", "Kill ws listener")
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
// API: /api/logout [POST]
// =============================================================================

// Request is an empty object

type APIPostLogoutRes struct {
  Error string
  Status string
}

func (s *StatusServer) logoutResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "POST":
    var res APIPostLogoutRes

    if err :=  s.cloud.Logout(); err != nil {
      config.Log(config.LOG_ERROR, "ss: ", err.Error())
      res = APIPostLogoutRes{Error: "No user data detected. User is already logged out.", Status: "error"}
    } else {
      s.initTemplates(TMPL_PATH)
      res = APIPostLogoutRes{Error: "", Status: "OK"}
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
// API: /api/output [POST]
// =============================================================================

type APIPostOutputReq struct {
  Address string
  Method string
}

type APIPostOutputRes struct {
  Error string
  Status string
}

type APIGetOutputRes struct {
  Outputs []string
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
    store := s.cloud.GetStore()

    if obj.Method == "delete" {
      config.Log(config.LOG_INFO, "ss: ", "Removing output address:", obj.Address)
      store.DelOutput(obj.Address)
      err = fmulink.Outputs.Remove(obj.Address)
    } else {
      config.Log(config.LOG_INFO, "ss: ", "Adding output address:", obj.Address)
      err = fmulink.Outputs.Add(obj.Address)
      store.SetOutput(obj.Address)
    }

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

  case "GET":
    store := s.cloud.GetStore()

    res := APIGetOutputRes{
      Outputs: store.GetOutput(),
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
  Step        string  `json:"step"`
  Error       string  `json:"error"`
}

const (
  SETUP_STEP_INITIAL = "setupInitial"
  SETUP_STEP_WIFICOMPLETE = "setupWifi"
  SETUP_STEP_DSSCOMPLETE = "setupDss"
)

func (s *StatusServer) setupResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "GET":
    var obj APIGetSetUpRes

    store := s.cloud.GetStore()
    storeStep := store.Get("step")

    if storeStep != "" {
      obj.Step = storeStep
    } else {
      // assume wifi is completed
      obj.Step = SETUP_STEP_WIFICOMPLETE
    }

    switch obj.Step {
    case SETUP_STEP_INITIAL: // wifi setup
      config.Log(config.LOG_INFO, "[SETUP] INITIAL SETUP PHASE.")
      // do nothing, in the initial setup phase
    case SETUP_STEP_WIFICOMPLETE:
      config.Log(config.LOG_INFO, "[SETUP] DS CLOUD SETUP PHASE.")
      // check wifi
      if ip, _, err := checkIP(); err != nil {
        obj.Error = err.Error()
      } else if ip { // connected. Load next step.
        if supp, err := isSupplicant(); err != nil { // supplicant mode means we can go to step 2.
          obj.Error = err.Error()
        } else if supp {
          config.Log(config.LOG_INFO, "In supplicant with an IP, no need to do anything.")
        } else {
          config.Log(config.LOG_INFO, "Not in supplicant mode, going to wifi setup.")
          obj.Step = SETUP_STEP_INITIAL
        }
      } else { // not connected.
        config.Log(config.LOG_INFO, "Ip is none, going to wifi setup.")

        obj.Step = SETUP_STEP_INITIAL
        store.Set("step", obj.Step)

        if supp, err := isSupplicant(); err != nil {
          obj.Error = err.Error()
        } else if supp {
          go func() {
            runEdisonCmd("--enableOneTimeSetup")
            os.Exit(0)
          }()
        }
      }

    case SETUP_STEP_DSSCOMPLETE:
      config.Log(config.LOG_INFO, "[SETUP] SETUP COMPLETE.")
      // do nothing, dss login successful, render regular page
    default:
      config.Log(config.LOG_INFO, "[SETUP] INVALID PHASE. GOING TO INITIAL.")
      obj.Step = SETUP_STEP_INITIAL
    }

    store.Set("step", obj.Step)

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

    e := store.Get("email")
    p := store.Get("pass")

    if e != "" || p != "" {
      err = fmt.Errorf("Already activated.")
    } else {
      err = store.Set("email", obj.Email)
      err = store.Set("pass", obj.Password)
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
        if err := store.Set("step", SETUP_STEP_DSSCOMPLETE); err != nil {
          res = APIPostSetupRes{Error: err.Error(), Status: "error"}
        } else {
          res = APIPostSetupRes{Error: "", Status: "OK"}
          s.initTemplates(TMPL_PATH)
        }
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
// API: /api/sensor
// =============================================================================

type APISensorRes struct {
  Status string `json:"status"`
  Error string `json:"error"`
}

func (s *StatusServer) sensorResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "POST":
    var obj cloudlink.SensorReq
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&obj)
    if err != nil {
      panic(err)
    }

    var res APISensorRes
    s.cloud.SendSensor(&obj)
    res = APISensorRes{Error: "", Status: "OK"}

    if data, err := json.Marshal(res); err != nil {
      panic(err)
    } else {
      if _ , err := w.Write(data); err != nil {
        panic(err)
      }
    }
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
  Error     string  `json:"error"`
}

func (s *StatusServer) apsResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "POST":
    var obj APIPostApsReq
    updateWifi := false
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

    var namesMap map[string]string
    namesMap, err = getNames()

    // Only change name if it is different.
    if (namesMap != nil && namesMap["ssid"] != name) {
      err = setName(name)

      // verify the change worked.
      namesMap, err = getNames()
    }

    // Error check protos
    switch obj.Protocol {
    case "OPEN":
      // no password
      updateWifi = true
    case "WEP":
      if (len(obj.Password) != 5 || len(obj.Password) != 13) {
        err = fmt.Errorf("Network password must be either 5 or 13 characters in length.")
      } else {
        updateWifi = true
      }
    case "WPA-PSK":
      if (len(obj.Password) < 8 || len(obj.Password) > 63) {
        err = fmt.Errorf("Network password must be between 8 and 63 characters in length.")
      } else {
        updateWifi = true
      }
    default:
      err = fmt.Errorf("Invalid or unsupported network protocol.")
    }


    var res *APIPostApsRes

    if err != nil {
      res = &APIPostApsRes{Error: err.Error(),}
    } else {
      res = &APIPostApsRes{obj.Ssid, namesMap["ssid"], "",}
    }

    if data, err := json.Marshal(res); err != nil {
      panic(err)
    } else {
      if _ , err := w.Write(data); err != nil {
        panic(err)
      } else if (updateWifi) {
        // Update wifi after the response to ensure user gets a response.
        go func() {
          config.Log(config.LOG_DEBUG, "ss:  updating wifi")
          switch obj.Protocol {
          case "OPEN":
            runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid)
          case "WEP":
            runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid, obj.Password)
          case "WPA-PSK":
            runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid, obj.Password)
          case "WPA_EAP":
            runEdisonCmd("--changeWiFi", obj.Protocol, obj.Ssid, obj.Username, obj.Password)
          }

          config.Log(config.LOG_INFO, "ss:  Checking IP...")
          time.Sleep(30 * time.Second)
          if ip, _, err := checkIP(); err != nil {
            runEdisonCmd("--enableOneTimeSetup")
          } else if !ip {
            runEdisonCmd("--enableOneTimeSetup")
          }

          config.Log(config.LOG_INFO, "ss:  rebooting DS Link...")
          store := s.cloud.GetStore()
          store.Set("step", SETUP_STEP_WIFICOMPLETE)
          os.Exit(0)
        }()
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
