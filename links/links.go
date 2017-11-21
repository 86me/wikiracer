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
)

const (
    apiEndpoint = "http://en.wikipedia.org/w/api.php"
    userAgent= "wikiracer/0.86 (http://github.com/86me/wikiracer); egon@hyszczak.net"

    namespace = "0|14|100"
)

var (
    client = &http.Client{ Timeout: 5 * time.Second }

    // Ignore uninteresting or "boring" term relationships
    boring = map[string]bool {
        "Biblioteca Nacional de España":                    true,
        "Bibliothèque nationale de France":                 true,
        "Digital object identifier":                        true,
        "Integrated Authority File":                        true,
        "LIBRIS":                                           true,
        "CNN":                                              true,
        "Wayback Machine":                                  true,
        "Library of Congress Control Number":               true,
        "MusicBrainz":                                      true,
        "AllMusic":                                         true,
        "Billboard (magazine)":                             true,
        "List of Rock and Roll Hall of Fame inductees":     true,
        "National Diet Library":                            true,
        "Virtual International Authority File":             true,
    }

    boring_regex = []string {
        "^Category:Articles with unsourced.*$",
        "^International Standard.*$",
        "^National Library of.*$",
        "^PubMed.*$",
        "^DMOZ$",
    }

    boring_regex_pattern = `(` + strings.Join(boring_regex, "|") + `)`

)

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
        //"exintro":      {""},
        //"excontinue":   {""},
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
    // Check against boring titles and discard matches
    if boring[from] || boring[to] {
        return
    }

    // Check against boring regular expressions and discard matches
    r, _ := regexp.Compile(boring_regex_pattern)
    if r.Match([]byte(from)) || r.Match([]byte(to)) {
        return
    }

    // The API can return pages that link to themselves. We should ignore them.
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

// LinksFrom takes one or more Wikipedia page titles and returns a channel that
// will receive one or more Links objects, each containing partial or full
// mappings of linked page to source page. The channel will be closed after all
// results have been fetched.
func LinksHere(titles []string) chan Links {
    return allLinks("lh", "linkshere", titles)
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
//
//   {
//     "continue": {
//       "{subkey}": "736|0|Action-angle_variables"
//     }
//   }
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
//
//   {
//     ...
//     "query": {
//       "pages": {
//         "15580374": {
//           "title": "Albert Einstein",
//           "{subkey}":[
//             { "title": "2dF Galaxy Redshift Survey" },
//             ...
//           ]
//         },
//         ...
//       }
//     }
//   }
func extractLinks(data map[string]interface{}, subkey string) Links {
    links := Links{}

    //fmt.Println("query: ", data["query"])

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

    //fmt.Println("links: ", links)

    return links
}
