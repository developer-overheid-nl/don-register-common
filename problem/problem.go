package problem

import "net/http"

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

type Problem struct {
	Title  string        `json:"title"`
	Status int           `json:"status"`
	Errors []ErrorDetail `json:"errors,omitempty"`
}

func (e Problem) Error() string { return e.Title }

func New(status int, title string, details ...ErrorDetail) Problem {
	return Problem{
		Status: status,
		Title:  title,
		Errors: details,
	}
}

func NewBadRequest(title string, details ...ErrorDetail) Problem {
	return New(http.StatusBadRequest, title, details...)
}

func NewNotFound(title string, details ...ErrorDetail) Problem {
	return New(http.StatusNotFound, title, details...)
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
