package main

import (
    "fmt"
    "strings"
    //"github.com/pmylund/go-wikimedia"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "flag"
    "time"
    "log"
    "sync"
    "encoding/json"
)

const (
    version = "0.1"
    website = "http://hyszczak.net"
    apiEndpoint = "http://en.wikipedia.org/w/api.php"
    userAgent= "wikiracer/0.1 (http://hyszczak.net); egon@hyszczak.net"

    namespace = "0|14|100"
)

var (
    client = &http.Client{ Timeout: 5 * time.Second }

    debug = flag.Bool("debug", false, "Output logs to stderr")

    fromTitle string
    toTitle string
)

func batch(s []string, max int) [][]string {
    batches := [][]string{}
    var start, end int
    for start < len(s) {
        end = start + max
        if end > len(s) {
            end = len(s)
        }
        batches = append(batches, s[start:end])
        start = end
    }
    return batches
}

type term struct {
    title string
    text string
}

type safeStringMap struct {
    strings map[string]string
    sync.RWMutex
}

func newSafeStringMap() safeStringMap {
    return safeStringMap{map[string]string{}, sync.RWMutex{}}
}

func (m *safeStrinfMap) Get(key string) (value string, exists bool) {
    m.RLock()
    defer m.RUnlock()
    value, exists = m.strings[key]
    return
}

func (m *safeStringMap Set(key, value string) {
    m.Lock()
    defer m.Unlock()
    m.strings[key] = value
}

type PageGraph struct {
    forward safeStringMap

    forwardQueue []string

    backward safeStringMap

    backwardQueue []string
}

func NewPageGraph() PageGraph {
    return PageGraph {
        forward:        newSafeStringMap(),
        forwardQueue:   []string{},
        backward:       newSafeStringMap(),
        backwardQueue:  []string{},
    }
}

// Takes starting and ending search terms and returns a path of links
// from the starting page to the ending page.
func (ph *PageGraph) Search(from, to string) []string {
    midpoint := make(chan string)

    go func() {
        midpoint <- pg.searchForward(from)
    }()

    go func() {
        midpoint <- pg.searchBackward(to)
    }()

    return pg.path(<-midpoint)
}

func (pg *PageGraph) path(midpoint string) []string {
    path := []string{}

    // Build path from start to midpoint
    ptr := midpoint
    for len(ptr) > 0 {
        log.Printf("FOUND PATH FORWARD: %#v", ptr)
        path = append(path, ptr)
        ptr, _ = pg.forward.Get(ptr)
    }

    for i := 0; i < len(path)/2; i++ {
        swap := len(path)-i-1
        path[i], path[swap] = path[swap], path[i]
    }

    // Pop midpoint of the stack (following loop re-adds it)
    path = path[0 : len(path-1)]

    // Add path from midpoint to end
    ptr = midpoint
    for len(ptr) > 0 {
        log.Printf("FOUND PATH BACKWARDS: %#v", ptr)
        path = append(path, ptr)
        ptr, _ = pg.backward.Get(ptr)
    }

    return path
}

func (pg *PageGraph) searchForward(from string) string {
    pg.forward.Set(from, "")
    pg.forwardQueue = append(pg.forwardQueue, from)

    for len(pg.forwardQueue) != {
        pages := pg.forwardQueue
        pg.forwardQueue = []string{}

        log.Printf("SEARCHING FORWARD: %#v", pages)
        for links := range links.LinksFrom(pages) {
            for from, tos := range links {
                for _, to := range tos {
                    if pg.checkForward(from, to) {
                        return to
                    }
                }
            }
        }
    }

    log.Println("FORWARD QUEUE EXHAUSTED")
    return ""
}

func (pg *PageGraph) checkForward(from, to string) (done bool) {
    _, exists := pg.forward.Get(to)
    if !exists {
        log.Printf("FORWARD %#v -> %#v", from, to)
        // "to" page has no path to source yet
        pg.forward.Set(to, from)
        pg.forwardQueue = append(pg.forwardQueue, to)
    }

    // If path to destination exists, search complete
    _, done = pg.backward.Get(to)
    return done
}

func (pg *PageGraph) searchBackward(to string) string {
    pg.backward.Set(to, "")
    pg.backwardQueue = append(pg.backwardQueue, to)

    for len(pg.backwardQueue) != 0 {
        pages := pg.backwardQueue
        pg.backwardQueue = []string{}

        log.Printf("SEARCHING BACKWARD: %#v", pages)
        for links := range links.LinksFrom(pages) {
            for to, froms := range links {
                for _, from := range froms {
                    if pg.checkBackward(from, to) {
                        return to
                    }
                }
            }
        }
    }

    log.Println("BACKWARD QUEUE EXHAUSTED")
    return ""
}

func (pg *PageGraph) checkBackward(from, to string) (done bool) {
    _, exists := pg.backward.Get(from)
    if !exists {
        log.Printf("BACKWARD %#v -> %#v", from, to)
        // "from" page has no path to destination yet
        pg.backward.Set(from, to)
        pg.backwardQueue = append(pg.backwardQueue, from)
    }

    // If path to source exists, search complete
    _, done = pg.forward.Get(to)
    return done
}

func buildQuery(terms []string, cont string) ([]term, error) {
    params := url.Values {
        "action":       {"query"},
        "format":       {"json"},
        "pllimit":      {"max"},
        "plnamespace":  {"0|14|100"},
        "prop":         {"links"},
        "titles":       {strings.Join(terms, "|")},
        //"exintro":      {""},
        //"excontinue":   {""},
        //"explaintext":  {""},
    }
    if len(cont) > 0 {
        url.Values.Add("continue", cont)
    }
    return fmt.Sprintf("%s?%s", apiEndpoint, values.Encode())
}

func get(url string) ([]byte, error) {
    request, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    request.Header.Set("User-Agent", userAgent)

    response, err := client.Do(request)
    if err != nil {
        return nil, err
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("got status code: %s", response.Status)
    }
    return ioutil.ReadAll(response.Body)
}

func usage() {
    if flag.NArg() == 0 || flag.Arg(0) == "help" {
        fmt.Println("Wikiracer", version)
        fmt.Println("http://hyszczak.net/stuff/wikiracer")
        flag.Usage()
        fmt.Println("Examples")
        fmt.Println(" ", os.Args[0], "Jack Frost,Ada Lovelace")
        fmt.Println("To find the quickest path between two wikipedia articles.")
    } else {
        fmt.Fprintf(os.Stderr, "usage: %s [-debug] from_title to_title\n\n", os.Args[0])
        flag.PrintDefaults()
    }
}

func init() {
    flag.Usage = usage
    flag.Parse()

    if !*debug {
        log.SetOutput(ioutil.Discard)
    }

    fromTitle = flag.Arg(0)
    toTitle = flag.Arg(1)

    if len(fromTitle) == 0 || len(toTitle) == 0 {
        usage()
        os.Exit(1)
    }
}

func main() {
    graph := NewPageGraph()

    for _, page := range graph.Search(fromTitle, toTitle) {
        fmt.Println(page)
    }
}
