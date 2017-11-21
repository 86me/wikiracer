# WikiRacer

Given two Wikipedia pages, WikiRacer will attempt to find the shortest path
between the two using only links to other Wikipedia pages.

WikiRacer uses live data (via the MediaWiki API) and is extremely fast
by using a bi-directional, depth-first search algorithm.

```
% time ./wikirace Altoids Doorbell
Altoids
Cinnamon
China
Door
Doorbell
./wikirace Altoids Doorbell  0.11s user 0.03s system 17% cpu 0.776 total

% time ./wikirace "Preparation H" Pokémon
Preparation H
The New York Times
Chicago Sun-Times
Pokémon
./wikirace "Preparation H" Pokémon  0.10s user 0.03s system 17% cpu 0.718 total
```

## Building

WikiRacer has no external dependencies. Just fetch and build with: `go get
github.com/86me/wikiracer`

## Running

```
usage: ./wikirace [-debug] from_title to_title

  -debug
      print debugging log output to stderr
```

Example:

```
% wikirace "Mike Tyson" "Oceanography"
Mike Tyson
Alexander the Great
Aegean Sea
Oceanography
```

[good_target]: https://en.wikipedia.org/wiki/Wikipedia:Wikirace#Good_Target_Pages

## Limitations

* wikirace adheres to the [WikiMedia etiquette guide][etiquette] as faithfully
  as possible. To that end, it runs, at most, two simultaneous API requests to
  Wikipedia at a time.

* Wikipedia's API for fetching links from / to pages isn't as granular as the
  raw HTML, which can make it hard to exclude "boring" link paths. For example,
  many pages have an "[Authority control][auth_control]" block which has links
  to pages like "International Standard Book Number" which are linked to from
  other pages with "Authority control" sections. I've excluded as many of those
  as I could find.

[etiquette]: https://www.mediawiki.org/wiki/API:Etiquette
[auth_control]: https://en.wikipedia.org/wiki/Help:Authority_control

## Process

* **1 hour** - Researching possible strategies for building wikirace including
  tools, libraries, algorithms, and Wikipedia's MediaWiki API. For algorithms,
  the bi-directional, depth-first method immediately stood out to me as the way
  to go, with normal breadth-first/depth-first complexity being [O(b^d)], 
  executing two searches would reduce complexity to [O(b^(d/2))] for each search,
  bringing total complexity to [O(b^(d/2) + b^(d/2)], which is far less than the 
  vanilla BFS/DFS method.

* **1 hours** - Familiarizing myself with the MediaWiki API. I started with
  https://github.com/patrickmn/go-wikimedia which was far too basic for anything
  other than making initial test queries to see how the API responded to basic 
  requests. I studied a few different implementations of the MediaWiki API in 
  other languages, but decided to use go because I really enjoy writing go code, 
  I want to get more familiar with go internals, mutexes and concurrency patterns.
  I also really love the testing functionality of go and find it extremely useful.

* **2 hours** - Came across [Tyson Mote's][tysonmote] implementation that used
  the BDFS algoritm and used it as a frame of reference
[tysonmote]: https://github.com/tysonmote/wikirace

* **2 hours** - Converted the one-request-per-page API code to a batch model
  that would allow me to fetch (for example) the adjacent pages for a list of 50
  pages at one time rather than issuing an API request for each and every page.
  This model adheres to Wikipedia's published API limits.

* **2 hours** - Replaced unidirectional depth-first search with a bidirectional
  depth-first search. This is orders of magnitude faster in practice -- I'm able
  to find short paths between unrelated pages in around a second from my local
  machine. Car -> Petunia in 0.8s, Mike Tyson -> Carp in 0.7s, Pencil -> Calcium
  in 1.2s, Google -> Wheat in 0.7s, etc.

* **1 hour** - Writing README, tidying up some odds and ends, adding some
  documentation throughout.
