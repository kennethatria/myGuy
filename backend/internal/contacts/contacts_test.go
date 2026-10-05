package contacts

import (
	"encoding/json"
	"os"
	"testing"
)

func TestContainsSharedCases(t *testing.T) {
	raw, err := os.ReadFile("../../../shared/contact-filter-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases struct {
		Contacts []string `json:"contacts"`
		Clean    []string `json:"clean"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}

	for _, text := range cases.Contacts {
		if !Contains(text) {
			t.Errorf("expected contact detected in %q", text)
		}
	}
	for _, text := range cases.Clean {
		if Contains(text) {
			t.Errorf("unexpected contact detected in %q", text)
		}
	}
}
