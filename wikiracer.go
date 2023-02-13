package main

import (
    "fmt"
    "strings"
    "os"
    "flag"
    "log"
    "io/ioutil"
    "github.com/86me/wikiracer/net"
    "github.com/86me/wikiracer/links"
    "time"
)

var (
    debug = flag.Bool("debug", false, "Output logs to stderr")
    help = flag.Bool("help", false, "Additional help information")
    serve = flag.Bool("serve", false, "Run HTTP server")

    fromTitle string
    toTitle string
)

func usage() {
    if *help && len(flag.Arg(1)) == 0 {
        fmt.Println("Wikiracer", net.Version)
        fmt.Println(net.Website, "/segment/wikiracer")
        fmt.Println("Examples:")
        fmt.Println(" ", os.Args[0], "\"Robert Frost\" \"Ada Lovelace\"")
        fmt.Println(" ", os.Args[0], "\"Akira\" \"Ghost in the Shell\"")
        fmt.Println("To find the quickest path between two wikipedia articles.")
        fmt.Println(" ", os.Args[0], "-serve [address:port]")
        fmt.Println("To serve WikiRacer on HTTP [address:port]")
        os.Exit(1)
    } else {
        fmt.Fprintf(os.Stderr, "usage: %s [-debug] [-serve] \"from_title\" \"to_title\"\n\n", os.Args[0])
        flag.PrintDefaults()
    }
}

func init() {
    flag.Usage = usage
    flag.Parse()

    if !*debug {
      log.SetOutput(ioutil.Discard)
    } else {
      log.SetOutput(os.Stdout)
    }

    // Start HTTP service
    if *serve {
        wr := net.WikiRace{}
        wr.Initialize()
        port := flag.Arg(0)
        if len(port) > 0 {
            wr.Serve(flag.Arg(0))
        } else {
            wr.Serve(":8686") // default port
        }
    }

    // Parse command line arguments
    fromTitle = flag.Arg(0)
    toTitle = flag.Arg(1)

    // If empty print usage and exit
    if len(fromTitle) == 0 || len(toTitle) == 0 {
        usage()
        os.Exit(1)
    }

}

func main() {
    startTime := time.Now()

    // Run wikirace
    graph := links.NewPageGraph()
    var links[]string
    for _, page := range graph.Search(fromTitle, toTitle) {
        links = append(links, page)
    }
    fmt.Println(strings.Join(links, ` -> `))
    graph.Stop()

    fmt.Println("Elapsed time: ", time.Since(startTime))
}

