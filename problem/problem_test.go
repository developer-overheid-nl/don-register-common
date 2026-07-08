package problem

import (
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
			want: Problem{Status: http.StatusTeapot, Title: "teapot", Errors: []ErrorDetail{detail}},
		},
		{
			name: "NewBadRequest",
			got:  NewBadRequest("bad request", detail),
			want: Problem{Status: http.StatusBadRequest, Title: "bad request", Errors: []ErrorDetail{detail}},
		},
		{
			name: "NewNotFound",
			got:  NewNotFound("not found"),
			want: Problem{Status: http.StatusNotFound, Title: "not found"},
		},
		{
			name: "NewInternalServerError",
			got:  NewInternalServerError("internal"),
			want: Problem{Status: http.StatusInternalServerError, Title: "internal"},
		},
		{
			name: "NewForbidden",
			got:  NewForbidden("forbidden"),
			want: Problem{Status: http.StatusForbidden, Title: "forbidden"},
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
