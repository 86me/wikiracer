package main

import (
    "fmt"
    "os"
    "flag"
    "log"
    "sync"
    "io/ioutil"
    "wikiracer/links"
    "time"
)

const (
    version = "0.86"
    website = "http://hyszczak.net"
)

var (
    debug = flag.Bool("debug", false, "Output logs to stderr")

    fromTitle string
    toTitle string
)

type safeStringMap struct {
    strings map[string]string
    sync.RWMutex
}

func newSafeStringMap() safeStringMap {
    return safeStringMap{map[string]string{}, sync.RWMutex{}}
}

func (m *safeStringMap) Get(key string) (value string, exists bool) {
    m.RLock()
    defer m.RUnlock()
    value, exists = m.strings[key]
    return
}

func (m *safeStringMap) Set(key, value string) {
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
func (pg *PageGraph) Search(from, to string) []string {
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
    path = path[0 : len(path)-1]

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

    for len(pg.forwardQueue) != 0 {
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

func usage() {
    if flag.Arg(0) == "help" && len(flag.Arg(1)) == 0 {
        fmt.Println("Wikiracer", version)
        fmt.Println("http://hyszczak.net/stuff/wikiracer")
        flag.Usage()
        fmt.Println("Examples")
        fmt.Println(" ", os.Args[0], "Jack Frost,Ada Lovelace")
        fmt.Println("To find the quickest path between two wikipedia articles.")
        os.Exit(1)
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
    startTime := time.Now()
    graph := NewPageGraph()

    for _, page := range graph.Search(fromTitle, toTitle) {
        fmt.Println(page)
    }

    fmt.Println("Elapsed time: ", time.Since(startTime))
}
