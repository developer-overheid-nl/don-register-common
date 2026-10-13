package problem

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestProblemConstructors(t *testing.T) {
	detail := ErrorDetail{In: "body", Location: "name", Code: "required", Detail: "name is required"}

	tests := []struct {
		name string
		got  Problem
		want Problem
	}{
		{
			name: "New",
			got:  New(http.StatusTeapot, "teapot", detail),
			want: Problem{Status: http.StatusTeapot, Title: "teapot", Detail: "name is required", Errors: []ErrorDetail{detail}},
		},
		{
			name: "NewBadRequest",
			got:  NewBadRequest("bad request", detail),
			want: Problem{Status: http.StatusBadRequest, Title: "bad request", Detail: "name is required", Errors: []ErrorDetail{detail}},
		},
		{
			name: "NewNotFound",
			got:  NewNotFound("not found"),
			want: Problem{Status: http.StatusNotFound, Title: "not found", Detail: "not found"},
		},
		{
			name: "NewInternalServerError",
			got:  NewInternalServerError("internal"),
			want: Problem{Status: http.StatusInternalServerError, Title: "internal", Detail: "internal"},
		},
		{
			name: "NewForbidden",
			got:  NewForbidden("forbidden"),
			want: Problem{Status: http.StatusForbidden, Title: "forbidden", Detail: "forbidden"},
		},
		{
			name: "NewMethodNotAllowed",
			got:  NewMethodNotAllowed("method not allowed"),
			want: Problem{Status: http.StatusMethodNotAllowed, Title: "method not allowed", Detail: "method not allowed"},
		},
		{
			name: "NewWithDetail",
			got:  NewWithDetail(http.StatusBadRequest, "bad request", "the name is missing", detail),
			want: Problem{Status: http.StatusBadRequest, Title: "bad request", Detail: "the name is missing", Errors: []ErrorDetail{detail}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Fatalf("problem = %#v, want %#v", tt.got, tt.want)
			}
			if tt.got.Error() != tt.want.Title {
				t.Fatalf("Error() = %q, want %q", tt.got.Error(), tt.want.Title)
			}
		})
	}
}

func TestDetailJoinsErrorDetails(t *testing.T) {
	got := New(http.StatusBadRequest, "bad request",
		ErrorDetail{Location: "name", Detail: "name is required"},
		ErrorDetail{Location: "url", Detail: " "},
		ErrorDetail{Location: "url", Detail: "url is invalid"},
	)
	if got.Detail != "name is required; url is invalid" {
		t.Fatalf("Detail = %q, want the error details joined with \"; \"", got.Detail)
	}
	if empty := NewWithDetail(http.StatusNotFound, "not found", ""); empty.Detail != "not found" {
		t.Fatalf("Detail = %q, want the title as fallback", empty.Detail)
	}
}

func TestDetailIsAlwaysSerialized(t *testing.T) {
	body, err := json.Marshal(NewNotFound("not found"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"title":"not found","status":404,"detail":"not found"}` {
		t.Fatalf("json = %s", body)
	}
}

func TestErrorDetailsFromInvalidParamsUsesInvalidParams(t *testing.T) {
	got := ErrorDetailsFromInvalidParams([]InvalidParam{
		{Name: "name", Reason: "name is required"},
		{Name: "url", Reason: "url is invalid"},
	}, "fallback", "query", "filter", "invalid")

	want := []ErrorDetail{
		{In: "body", Location: "name", Code: "name", Detail: "name is required"},
		{In: "body", Location: "url", Code: "url", Detail: "url is invalid"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ErrorDetailsFromInvalidParams() = %#v, want %#v", got, want)
	}
}

func TestErrorDetailsFromInvalidParamsFallback(t *testing.T) {
	got := ErrorDetailsFromInvalidParams(nil, "invalid page", "query", "page", "invalid")
	want := []ErrorDetail{{In: "query", Location: "page", Code: "invalid", Detail: "invalid page"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback details = %#v, want %#v", got, want)
	}

	if details := ErrorDetailsFromInvalidParams(nil, "", "query", "page", "invalid"); details != nil {
		t.Fatalf("empty fallback details = %#v, want nil", details)
	}
}
