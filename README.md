# WikiRacer

Given two Wikipedia pages, WikiRacer will attempt to find the shortest path
between the two using only links to other Wikipedia pages.

WikiRacer uses live data (via the MediaWiki API) and is extremely fast
by using a bi-directional, depth-first search algorithm.

```
$ time ./wikiracer Pleiades OpenBSD
Pleiades
Bible
Système universitaire de documentation
Bill Joy
OpenBSD
Elapsed time:  7.120570172s
./wikiracer Pleiades OpenBSD  0.49s user 0.04s system 7% cpu 7.132 total

$ time ./wikiracer "Jim Beam" "King George"
Jim Beam
Kentucky
Letters patent
George IV of the United Kingdom
King George
Elapsed time:  4.672853825s
./wikiracer "Jim Beam" "King George"  0.32s user 0.03s system 7% cpu 4.684 total
```

## Building

WikiRacer has no external dependencies. Just fetch and build with: `go get
github.com/86me/wikiracer`

## Running

```
usage: ./wikiracer [-debug] "from_title" "to_title"

  -debug
        Output logs to stderr
  -help
        Additional help information
  -serve
        Run HTTP server
```

Example:

```
$ ./wikiracer "Ada Lovelace" "Robert Frost"
Ada Lovelace
Artificial intelligence
Dartmouth College
Robert Frost
Elapsed time:  3.033301067s
```

## Limitations

* wikirace adheres to the [WikiMedia etiquette guide][etiquette] as faithfully
  as possible. To that end, it runs, at most, two simultaneous API requests to
  Wikipedia at a time.

[etiquette]: https://www.mediawiki.org/wiki/API:Etiquette

## Process

* **1 hour** - Researching possible strategies for building `wikiracer` including
  tools, libraries, algorithms, and Wikipedia's MediaWiki API. For 
  [algorithms][algorithms], the bi-directional, depth-first method immediately 
  [stood out][geeksforgeeks] to me as the way to go, with normal 
  breadth-first/depth-first complexity being [O(b^d)], executing two searches 
  would reduce complexity to [O(b^(d/2))] for each search, bringing total 
  complexity to [O(b^(d/2) + b^(d/2)], which is far less than the vanilla 
  BFS/DFS method.

[algorithms]: https://www.ics.uci.edu/~rickl/courses/cs-171/cs171-lecture-slides/cs-171-03-UninformedSearch.pdf
[geeksforgeeks]: http://www.geeksforgeeks.org/bidirectional-search/

* **1 hour** - [Familiarizing myself][cs-cornell] with the MediaWiki API. 
  I started with [patrickmn's go-wikimedia][go-wikimedia] implementation, which 
  was far too basic for anything other than making initial test queries to see 
  how the API responded to basic requests. I studied a few different 
  implementations of the MediaWiki API in other languages, but decided to use 
  go because I really enjoy writing go code, I want to get more familiar with 
  go internals, mutexes and concurrency patterns. I also really love the testing
  functionality of go and find it extremely powerful.

[cs-cornell]: http://www.cs.cornell.edu/~wdtseng/icpc/notes/bt3.pdf
[go-wikimedia]: https://github.com/patrickmn/go-wikimedia

* **2 hours** - Started fleshing out the base functionality. Came across 
  [Tyson Mote's][tysonmote] implementation that used the BDFS algoritm and used 
  it as a frame of reference. I especially appreciated the RWMutex concurrency 
  pattern. Implemented a batch request model to keep overall API requests to a 
  minimum. This model adheres to Wikipedia's published API limits. Added regular
  expressions matching to the boring links to better filter out unwanted paths.

[tysonmote]: https://github.com/tysonmote/wikirace

* **1 hour** - Implemented a REST API using [gorrilla/mux][gorilla/mux]. I could
  have just used the native go net/http package, but gorilla offered some helpful
  abstractions (mux/router). The REST API is accessible by running `wikiracer` 
  with the `-serve` switch. It binds to port 8686 by default. The port and address
  can be passed in as a parameter to the `-serve` switch. eg:
    `wikiracer -serve 0.0.0.0:4040`

[gorilla/mux]: https://github.com/gorilla/mux

* **2 hours** - Testing, writing tests, writing README, testing, more testing.
