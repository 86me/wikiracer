package net

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "regexp"
)

var wr WikiRace

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
    rr := httptest.NewRecorder()
    wr.Router.ServeHTTP(rr, req)

    return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
    if expected != actual {
        t.Errorf("Expected response code %d. Got %d\n", expected, actual)
    }
}

func TestGetHelp(t *testing.T) {
    wr = WikiRace{}
    wr.Initialize()

    req, _ := http.NewRequest("GET", "/", nil)
    response := executeRequest(req)

    checkResponseCode(t, http.StatusOK, response.Code)

    if status := response.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    expected := `{"Body":"Example usage: /Ada Lovelace/Susan B. Anthony"}`
    if response.Body.String() != expected {
        t.Errorf("handler returned unexpected body: got %v want %v", response.Body.String(), expected)
    }
}

func TestRunRace(t *testing.T) {
    wr = WikiRace{}
    wr.Initialize()

    req, _ := http.NewRequest("GET", "/Ada Lovelace/Susan B. Anthony", nil)
    response := executeRequest(req)

    checkResponseCode(t, http.StatusOK, response.Code)

    if status := response.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    // Remove elapsed time to help match response. Will sometimes fail depending
    // on path found. Could be reworked to a more stable page path.
    re := regexp.MustCompile(`Elapsed time: [0-9].[0-9]*s`)

    expected := `<h1>WikiRacer 0.86</h1><br/>
                <h2>From Ada Lovelace to Susan B. Anthony:</h2>
                <p>Ada Lovelace &rarr; Artificial intelligence &rarr; Albert Einstein &rarr; Susan B. Anthony</p><br/>
                <small></small>`

    response_regex := re.ReplaceAllString(response.Body.String(), "")
    if response_regex != expected {
        t.Errorf("Handler returned unexpected body: got '%v' want '%v'", response_regex, expected)
    }
}
