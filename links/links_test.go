package links

import (
  "encoding/json"
  "reflect"
  "strings"
  "testing"
)

const (
  partialLinksJSON = `{
    "continue": {
        "plcontinue": "39027|0|AAU_Junior_Olympic_Games",
        "continue": "||"
    },
    "query": {
        "pages": {
            "7365423": {
                "pageid": 7365423,
                "ns": 0,
                "title": "Tryall Golf Club"
            },
            "39027": {
                "pageid": 39027,
                "ns": 0,
                "title": "Mike Tyson",
                "links": [
                    {
                        "ns": 0,
                        "title": "1984 Summer Olympics"
                    },
                    {
                        "ns": 0,
                        "title": "20/20 (US television show)"
                    },
                    {
                        "ns": 0,
                        "title": "2009 Golden Globe Awards"
                    }
                ]
            }
        }
    },
    "limits": {
        "links": 3
    }
  }`
)

func TestLinksResponse_UnmarshalJSON(t *testing.T) {
  resp := linksResponse{prefix: "pl", prop: "links"}
  err := json.Unmarshal([]byte(partialLinksJSON), &resp)
  if err != nil {
    t.Fatal(err)
  }

  if resp.Continue != "39027|0|AAU_Junior_Olympic_Games" {
    t.Errorf("unexpected continute: %#v", resp.Continue)
  }

  expectLinks := Links{
    "Mike Tyson": []string{
      "1984 Summer Olympics",
      "20/20 (US television show)",
      "2009 Golden Globe Awards",
    },
  }

  if !reflect.DeepEqual(expectLinks, resp.Links) {
    t.Errorf("expected: %#v\ngot: %#v", expectLinks, resp.Links)
  }
}

func TestBatch(t *testing.T) {
  tests := []struct {
    given  []string
    size   int
    expect [][]string
  }{
    {[]string{}, 3, [][]string{}},
    {[]string{"a"}, 3, [][]string{{"a"}}},
    {[]string{"a", "b", "c"}, 3, [][]string{{"a", "b", "c"}}},
    {[]string{"a", "b", "c", "d"}, 3, [][]string{{"a", "b", "c"}, {"d"}}},
    {[]string{"a", "b", "c", "d", "e"}, 2, [][]string{{"a", "b"}, {"c", "d"}, {"e"}}},
  }

  for i, test := range tests {
    got := batch(test.given, test.size)
    if !reflect.DeepEqual(test.expect, got) {
      t.Errorf("tests[%d]: expected: %#v, got: %#v", i, test.expect, got)
    }
  }
}

func TestBuildQuery(t *testing.T) {
  url := buildQuery("xx", "titles", []string{"foo", "bar"}, "abc")

  params := []string{
    "prop=titles",
    "titles=foo%7Cbar",
    "xxcontinue=abc",
    "xxlimit=max",
    "xxnamespace=0",
  }
  for _, expected := range params {
    if !strings.Contains(url, expected) {
      t.Errorf("expected to find %#v in %#v", expected, url)
    }
  }
}
