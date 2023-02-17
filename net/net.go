package net

import (
  "fmt"
  "io"
  "os"
  "log"
  "strings"
  "time"
  "encoding/json"
  "github.com/86me/wikiracer/links"
  "net/http"
  "github.com/gorilla/mux"
)

const (
  Version = "0.86"
  Website = "http://hyszczak.net"
)

type WikiRace struct {
  Router  *mux.Router
}

func (wr *WikiRace) Initialize() {
  wr.Router = mux.NewRouter()
  wr.Router.HandleFunc("/", wr.GetHelp).Methods("GET")
  wr.Router.HandleFunc("/{from}", wr.RunRace).Methods("GET")
  wr.Router.HandleFunc("/{from}/{to}", wr.RunRace).Methods("GET")
}

func (wr *WikiRace) Serve(addr string) {
  fmt.Println("[WikiRacer] service running at", addr)
  log.Fatal(http.ListenAndServe(addr, wr.Router))
}

func (wr *WikiRace) GetHelp(w http.ResponseWriter, r *http.Request) {
  type Help struct {
    Body string
  }
  //responseJSON := Help{Body: "Example usage: /Ada Lovelace/Susan B. Anthony"}
  //respondWithJSON(w, http.StatusOK, responseJSON)
  responseHTML := `<h1>WikiRacer `+Version+`</h1><br/>
        <h2>Example usage:</h2>
        <p>http://localhost:8686/Ada Lovelace/Susan B. Anthony</p><br/>`
  respondWithHTML(w, http.StatusOK, responseHTML)
}

func (wr *WikiRace) RunRace(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  from := vars["from"]
  to := vars["to"]
  if len(from) == 0 || len(to) == 0 {
    respondWithError(w, http.StatusBadRequest, "Insufficient parameters")
    return
  }

  s := fmt.Sprintf("[%s] Remote request for %s -> %s\n", r.RemoteAddr, from, to)
  io.WriteString(os.Stdout, s)

  startTime := time.Now()
  // Run remote wiki race request
  graph := links.NewPageGraph()
  var links []string

  for _, page := range graph.Search(from, to) {
    links = append(links, page)
  }
  // Path found. Stop further depth searches
  graph.Stop()

  elapsed_time := time.Since(startTime)
  responseHTML := `<h1>WikiRacer `+Version+`</h1><br/>
            <h2>From `+from+` to `+to+`:</h2>
            <p>`+strings.Join(links, ` &rarr; `)+`</p><br/>
            <small>Elapsed time: `+elapsed_time.String()+`</small>`
  respondWithHTML(w, http.StatusOK, responseHTML)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
  respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithHTML(w http.ResponseWriter, code int, response string) {
  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  w.WriteHeader(code)
  w.Write([]byte(response))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
  response, _ := json.Marshal(payload)
  w.Header().Set("Content-Type", "application/json; charset=utf-8")
  w.WriteHeader(code)
  w.Write(response)
}
