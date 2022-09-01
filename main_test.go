package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
)

func TestAuth(t *testing.T) {
	resp, err := http.Get("http://localhost:6003/test")
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		res, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Fatal(string(res))
	}

	rs := []Dataset{}
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&rs)
	if err != nil {
		t.Fatal(err)
	}

	log.Println(rs)
}
