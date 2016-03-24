package setupServer

import (
  "bytes"
  "fmt"
  "html/template"
  "io"
  "encoding/json"
  "log"
  "net/http"
  "path/filepath"

  "fmulink"
)

var (
  uiData    []byte
  fileServer =  http.NewServeMux()
)

const (
  STATIC_PATH = "assets/public"
  TMPL_PATH = "assets/"
  SERVE_ADDRESS = ":8080"
)

func Init() {
  fileServer.Handle("/",          http.FileServer(http.Dir(STATIC_PATH)))
  http.HandleFunc("/",            rootHandler)
  http.HandleFunc("/api/setup",   setupResponse)
  http.HandleFunc("/api/aps",     apsResponse)
  http.HandleFunc("/api/fmu",     fmuResponse)

  if err := initSetup(TMPL_PATH); err != nil {
		panic(err)
	} else {
    log.Fatal(http.ListenAndServe(SERVE_ADDRESS, nil))
  }
}

func initSetup(root string) error {
  index := filepath.Join(root, "templates", "index.tmpl")
	ui, err := template.ParseFiles(index)
	if err != nil {
		return fmt.Errorf("parse index.tmpl: %v", err)
	}

  buffer := new(bytes.Buffer)
  templateData := struct {
    Title string
    HeaderText string
  }{"Luci Setup", "Hola from the Go server"}

  if err := ui.Execute(buffer, templateData); err != nil {
		return fmt.Errorf("render UI: %v", err)
	}
	uiData = buffer.Bytes()

  return nil
}

func rootHandler(w http.ResponseWriter, r* http.Request) {
  // Default error handler for bad web reqeusts. Sends out a 500.
  defer handler500(&w)

  // If it's root path, render index
  if r.URL.Path == "/" {
	  if err := renderUI(w); err != nil {
		  panic(err)
	  }
    return
  }

  // Handle static content
  fileServer.ServeHTTP(w, r)

  // Handle 404s
  // if r.URL.Path != "/" {
  //   http.Error(w, http.StatusText(404), 404)
  //   return
  // }
}

func renderUI(w io.Writer) error {
	if uiData == nil {
		panic("Could not load root template.")
	}
	_, err := w.Write(uiData)
	return err
}

func handler500(w* http.ResponseWriter) {
  if r := recover(); r != nil {
    http.Error(*w, http.StatusText(500) + ": " + r.(string), 500)
  }
}

//
// /api/setup
//

type APIPostSetupReq struct {
  Email string
  Password string
}

type APIPostSetupRes struct {
  Error string
}

func setupResponse(w http.ResponseWriter, r* http.Request) {
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

//
// /api/aps
//

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

func apsResponse(w http.ResponseWriter, r* http.Request) {
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

//
// /api/fmu
//

type APIGetFmuRes struct {
  Error     string
  Data      interface{}
}

func fmuResponse(w http.ResponseWriter, r* http.Request) {
  switch r.Method {
  case "GET":
    // TODO
    fmt.Println("TODO: GET FMU Status")
    fmt.Println(fmulink.GetStatus())
    fallthrough
  default:
    http.Error(w, http.StatusText(404), 404)
  }
}
