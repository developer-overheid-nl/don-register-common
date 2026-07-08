package pagination

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"
)

func TestNormalizeAppliesDefaultsAndMax(t *testing.T) {
	page, perPage := Normalize(0, 0, 25, 100)
	if page != 1 || perPage != 25 {
		t.Fatalf("Normalize(0, 0) = (%d, %d), want (1, 25)", page, perPage)
	}

	page, perPage = Normalize(3, 500, 25, 100)
	if page != 3 || perPage != 100 {
		t.Fatalf("Normalize with max = (%d, %d), want (3, 100)", page, perPage)
	}

	page, perPage = Normalize(2, 500, 25, 0)
	if page != 2 || perPage != 500 {
		t.Fatalf("Normalize without max = (%d, %d), want (2, 500)", page, perPage)
	}
}

func TestNewCalculatesNavigation(t *testing.T) {
	p := New(2, 10, 25)

	if p.CurrentPage != 2 || p.RecordsPerPage != 10 || p.TotalPages != 3 || p.TotalRecords != 25 {
		t.Fatalf("Pagination = %#v, want page 2 of 3 with 25 records", p)
	}
	if p.Next == nil || *p.Next != 3 {
		t.Fatalf("Next = %v, want 3", p.Next)
	}
	if p.Previous == nil || *p.Previous != 1 {
		t.Fatalf("Previous = %v, want 1", p.Previous)
	}

	empty := New(1, 10, 0)
	if empty.TotalPages != 0 || empty.Next != nil || empty.Previous != nil {
		t.Fatalf("empty pagination = %#v, want no pages or navigation", empty)
	}
}

func TestBuildLinkHeaderPreservesQueryAndUsesForwardedProto(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://api.example.test/items?sort=name&page=99", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "api.example.test"
	req.Header.Set("Forwarded-Proto", "https")

	next := 3
	p := Pagination{
		CurrentPage:    2,
		RecordsPerPage: 25,
		TotalPages:     4,
		TotalRecords:   90,
		Previous:       intPtr(1),
		Next:           &next,
	}

	got := BuildLinkHeader(req, p)
	want := `<https://api.example.test/items?page=1&perPage=25&sort=name>; rel="first", <https://api.example.test/items?page=1&perPage=25&sort=name>; rel="prev", <https://api.example.test/items?page=2&perPage=25&sort=name>; rel="self", <https://api.example.test/items?page=3&perPage=25&sort=name>; rel="next", <https://api.example.test/items?page=4&perPage=25&sort=name>; rel="last"`
	if got != want {
		t.Fatalf("BuildLinkHeader() = %q, want %q", got, want)
	}
}

func TestBuildLinkHeaderUsesTLSAndHandlesMissingInput(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://api.example.test/items", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "api.example.test"
	req.TLS = &tls.ConnectionState{}

	got := BuildLinkHeader(req, New(1, 10, 10))
	want := `<https://api.example.test/items?page=1&perPage=10>; rel="first", <https://api.example.test/items?page=1&perPage=10>; rel="self", <https://api.example.test/items?page=1&perPage=10>; rel="last"`
	if got != want {
		t.Fatalf("BuildLinkHeader() = %q, want %q", got, want)
	}

	if BuildLinkHeader(nil, New(1, 10, 10)) != "" {
		t.Fatal("BuildLinkHeader(nil, p) should be empty")
	}
	if BuildLinkHeader(req, Pagination{}) != "" {
		t.Fatal("BuildLinkHeader(req, empty pagination) should be empty")
	}
}

func TestSetHeadersIncludesPaginationAndLinkHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://api.example.test/items", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "api.example.test"

	headers := map[string]string{}
	SetHeaders(req, func(key, val string) {
		headers[key] = val
	}, New(1, 10, 12))

	want := map[string]string{
		"Total-Count":  "12",
		"Total-Pages":  "2",
		"Per-Page":     "10",
		"Current-Page": "1",
		"Link":         `<http://api.example.test/items?page=1&perPage=10>; rel="first", <http://api.example.test/items?page=1&perPage=10>; rel="self", <http://api.example.test/items?page=2&perPage=10>; rel="next", <http://api.example.test/items?page=2&perPage=10>; rel="last"`,
	}
	if !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %#v, want %#v", headers, want)
	}
}

func intPtr(v int) *int {
	return &v
}
