package links

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "time"
    "strings"
    "log"
    "regexp"
    "sync"
)

const (
    apiEndpoint = "http://en.wikipedia.org/w/api.php"
    userAgent= "wikiracer/0.86 (http://github.com/86me/wikiracer); egon@hyszczak.net"

    /* https://en.wikipedia.org/wiki/Wikipedia:Namespace#Programming */
    namespace = "0|14|100" // main|category|portal
)

var (
    tr = &http.Transport{
        MaxIdleConns:       10,
        IdleConnTimeout:    30 * time.Second,
        DisableCompression: true,
    }
    client = &http.Client{ Transport: tr, Timeout: 30 * time.Second }

    // Ignore uninteresting or "boring" term relationships
    boring_regex = []string {
        "^Category:Articles with unsourced.*$",
        "^Category:Redirects.*$",
        "^International Standard.*$",
        "^National Library of.*$",
        "^PubMed.*$",
        "^DMOZ$",
        "Integrated Authority File",
        "CNN",
        "JSTOR",
        "BIBSYS",
        "LIBRIS",
        "^OCLC$",
        "[Aa]bout.com",
        "[Ii][Mm][Dd][Bb]",
        "Wayback Machine",
        "National Diet Library",
        "Library of Congress Control Number",
        "Biblioteca Nacional de España",
        "Bibliothèque nationale de France",
    }
    boring_regex_pattern = `(` + strings.Join(boring_regex, "|") + `)`
)

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
        for links := range LinksFrom(pages) {
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
        for links := range LinksFrom(pages) {
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

// Returns the given slice as batches with a maximum size
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

func buildQuery(prefix, prop string, terms []string, cont string) (string) {
    params := url.Values {
        "action":       {"query"},
        "format":       {"json"},
        "prop":         {prop},
        "titles":       {strings.Join(terms, "|")},
        //"explaintext":  {""},
    }
    params.Add(fmt.Sprintf("%snamespace", prefix), namespace)
    params.Add(fmt.Sprintf("%slimit", prefix), "max")
    if len(cont) > 0 {
        params.Add(fmt.Sprintf("%scontinue", prefix), cont)
    }
    log.Printf("QUERY STRING: %s?%s", apiEndpoint, params.Encode())
    return fmt.Sprintf("%s?%s", apiEndpoint, params.Encode())
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

// Links is a mapping of directional page links using page titles.
type Links map[string][]string

func (pl Links) add(from, to string) {
    // Check against boring title expressions and discard matches
    boring := regexp.MustCompile(boring_regex_pattern)
    if boring.Match([]byte(from)) || boring.Match([]byte(to)) {
        return
    }

    // Ignore self-referential links
    if from == to {
        return
    }

    if _, ok := pl[from]; !ok {
        pl[from] = []string{}
    }
    pl[from] = append(pl[from], to)
}

// LinksFrom takes one or more Wikipedia page titles and returns a channel that
// will receive one or more Links objects, each containing partial or full
// mappings of page to linked page. The channel will be closed after all
// results have been fetched.
func LinksFrom(titles []string) chan Links {
    return allLinks("pl", "links", titles)
}

// allLinks batches API requests to fetch the maximum number of results allowed
// by Wikipedia and then sends Links objects containing those responses from
// Wikipedia on the returned channel.
func allLinks(prefix, prop string, titles []string) chan Links {
    c := make(chan Links)

    go func(prefix, prop string, titles []string) {
        // Holds Wikipedia's "continue" string if we have more results to fetch.
        // Set after the first request.
        var cont string

        // Wikipedia can batch process up to 50 page titles at a time.
        for _, titlesBatch := range batch(titles, 50) {
            // Continue paginating through results as long as Wikipedia is telling us
            // to continue.
            for i := 0; i == 0 || len(cont) > 0; i++ {
                queryURL := buildQuery(prefix, prop, titlesBatch, cont)
                body, err := get(queryURL)
                if err != nil {
                    // If Wikipedia returns an error, just panic instead of doing an
                    // exponential back-off.
                    panic(err)
                }

                // Parse the response.
                resp := linksResponse{prefix: prefix, prop: prop}
                err = json.Unmarshal(body, &resp)
                if err != nil {
                    panic(err)
                }

                c <- resp.Links
                cont = resp.Continue
            }
        }
        close(c)
    }(prefix, prop, titles)

    return c
}

// -- api response format

// linksResponse encapsulates Wikipedia's query API response with either
// "links" or "linkshere" properties enumerated.
type linksResponse struct {
    prefix   string
    prop     string
    Continue string
    Links    Links
}

func (r *linksResponse) UnmarshalJSON(b []byte) error {
    data := map[string]interface{}{}
    json.Unmarshal(b, &data)

    r.Continue = extractContinue(data, fmt.Sprintf("%scontinue", r.prefix))
    r.Links = extractLinks(data, r.prop)

    return nil
}

// extractContinue takes as input a Wikipedia API query response and returns
// the "continue" string. If no continue string is set, an empty string is
// returned.
func extractContinue(data map[string]interface{}, subkey string) string {
    if cont, ok := data["continue"]; ok {
        if contValue, ok := cont.(map[string]interface{})[subkey]; ok {
            return contValue.(string)
        }
    }
    return ""
}

// extractLinks takes as input a Wikipedia API query response with either
// "links" or "linkshere" properties enumerated for a set of pages and returns
// a complete Links representation of that response.
func extractLinks(data map[string]interface{}, subkey string) Links {
    links := Links{}

    query := data["query"].(map[string]interface{})
    pages := query["pages"].(map[string]interface{})
    for _, page := range pages {
        pageMap := page.(map[string]interface{})
        fromTitle := pageMap["title"].(string)
        linksSlice, ok := pageMap[subkey].([]interface{})
        if ok {
            for _, link := range linksSlice {
                linkMap := link.(map[string]interface{})
                links.add(fromTitle, linkMap["title"].(string))
            }
        }
    }
    return links
}
