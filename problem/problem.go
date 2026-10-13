package problem

import (
	"net/http"
	"strings"
)

type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type ErrorDetail struct {
	In       string `json:"in"`
	Location string `json:"location"`
	Code     string `json:"code"`
	Detail   string `json:"detail"`
}

// Problem is an RFC 9457 problem details response with the members the NL GOV
// API Design Rules require: status, title and detail.
type Problem struct {
	Title  string        `json:"title"`
	Status int           `json:"status"`
	Detail string        `json:"detail"`
	Errors []ErrorDetail `json:"errors,omitempty"`
}

func (e Problem) Error() string { return e.Title }

// New derives detail from the error details, joined with "; ", and falls back
// to the title when there are none.
func New(status int, title string, details ...ErrorDetail) Problem {
	return NewWithDetail(status, title, "", details...)
}

// NewWithDetail uses the given detail; an empty detail is derived as in New.
func NewWithDetail(status int, title, detail string, details ...ErrorDetail) Problem {
	if detail == "" {
		detail = detailFrom(title, details)
	}
	return Problem{
		Status: status,
		Title:  title,
		Detail: detail,
		Errors: details,
	}
}

func detailFrom(title string, details []ErrorDetail) string {
	parts := make([]string, 0, len(details))
	for _, d := range details {
		if msg := strings.TrimSpace(d.Detail); msg != "" {
			parts = append(parts, msg)
		}
	}
	if len(parts) == 0 {
		return title
	}
	return strings.Join(parts, "; ")
}

func NewBadRequest(title string, details ...ErrorDetail) Problem {
	return New(http.StatusBadRequest, title, details...)
}

func NewNotFound(title string, details ...ErrorDetail) Problem {
	return New(http.StatusNotFound, title, details...)
}

func NewMethodNotAllowed(title string, details ...ErrorDetail) Problem {
	return New(http.StatusMethodNotAllowed, title, details...)
}

func NewInternalServerError(title string, details ...ErrorDetail) Problem {
	return New(http.StatusInternalServerError, title, details...)
}

func NewForbidden(title string, details ...ErrorDetail) Problem {
	return New(http.StatusForbidden, title, details...)
}

func ErrorDetailsFromInvalidParams(params []InvalidParam, fallbackDetail, fallbackIn, fallbackLocation, fallbackCode string) []ErrorDetail {
	if len(params) == 0 {
		if fallbackDetail == "" {
			return nil
		}
		return []ErrorDetail{{
			In:       fallbackIn,
			Location: fallbackLocation,
			Code:     fallbackCode,
			Detail:   fallbackDetail,
		}}
	}
	out := make([]ErrorDetail, 0, len(params))
	for _, p := range params {
		out = append(out, ErrorDetail{
			In:       "body",
			Location: p.Name,
			Code:     p.Name,
			Detail:   p.Reason,
		})
	}
	return out
}
