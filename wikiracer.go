package main

import (
    "fmt"
    "strings"
    "github.com/pmylund/go-wikimedia"
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

func wikiSearch(terms []string, cont string) ([]term, error) {
    w, err := wikimedia.New("http://en.wikipedia.org/w/api.php")
    if err != nil {
        fmt.Println("Error initializing Wikimedia library:", err)
        os.Exit(1)
    }
    f := url.Values {
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
    result, err := w.Query(f)
    if err != nil {
        return nil, err
    }
    response := make([]term, len(result.Query.Pages))
    i := 0
    for _, v := range result.Query.Pages {
        response[i] = term{v.Title, v.Extract}
        i++
    }
    fmt.Println("result: ", result)
    fmt.Println("result.Query: ", result.Query)
    fmt.Println("response: ", response)
    return response, nil
}

func init() {
    flag.Parse()
}

func main() {
    if flag.NArg() == 0 || flag.Arg(0) == "help" {
        fmt.Println("Wikiracer", version)
        fmt.Println("http://hyszczak.net/stuff/wikiracer")
        flag.Usage()
        fmt.Println("Examples")
        fmt.Println(" ", os.Args[0], "Jack Frost,Ada Lovelace")
        fmt.Println("To find the quickest path between two wikipedia articles.")
    } else {
        input := strings.Split(strings.Join(flag.Args(), " "), ",")
        //fmt.Println("args: ", flag.Args())
        //fmt.Println("input: ", input)
        wikiSearch(input)
    }
}
