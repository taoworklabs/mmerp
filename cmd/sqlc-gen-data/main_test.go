package main

import (
	"os"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	in, err := os.ReadFile("testdata/request.json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/data.gen.go.golden")
	if err != nil {
		t.Fatal(err)
	}
	got, err := generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("output differs from golden\n--- got\n%s\n--- want\n%s", got, want)
	}
}

func TestRejectsUnknownCmd(t *testing.T) {
	in := `{"settings":{"queries":["internal/core/iam/queries.sql"]},
		"queries":[{"name":"CopyUsers","cmd":":execresult"}]}`
	_, err := generate([]byte(in))
	if err == nil || !strings.Contains(err.Error(), "iam.CopyUsers") {
		t.Fatalf("want error naming iam.CopyUsers, got %v", err)
	}
}

func TestRejectsBaseWithParams(t *testing.T) {
	in := `{"settings":{"queries":["internal/core/iam/queries.sql"]},
		"queries":[{"name":"UserList","cmd":":many","comments":[" data:base"],
		"columns":[{"name":"id"}],"params":[{"number":1,"column":{"name":"id"}}]}]}`
	_, err := generate([]byte(in))
	if err == nil || !strings.Contains(err.Error(), "iam.UserList") {
		t.Fatalf("want error naming iam.UserList, got %v", err)
	}
}
