package jobs

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestFailureCatalogueMatchesTheContract(t *testing.T) {
	raw, err := os.ReadFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Components struct {
			Schemas struct {
				FailureCode struct {
					Enum []string `json:"enum"`
				} `json:"FailureCode"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err = json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	want := spec.Components.Schemas.FailureCode.Enum
	got := FailureCodes()
	sort.Strings(want)
	sort.Strings(got)
	if len(want) == 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("server catalogue %v differs from the contract %v", got, want)
	}
	for _, code := range got {
		if RecommendedAction(code) == "" {
			t.Errorf("failure cause %s has no recommended action", code)
		}
	}
}
