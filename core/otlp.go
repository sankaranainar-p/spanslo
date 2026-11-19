package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// StatusCode is the OTLP span status code.
type StatusCode int

const (
	StatusUnset StatusCode = 0
	StatusOK    StatusCode = 1
	StatusError StatusCode = 2
)

// Status is a span status.
type Status struct {
	Code    StatusCode
	Message string
}

// Event is a span event. Attr values are string, int64, float64, bool, []any or map[string]any.
type Event struct {
	Name  string
	Time  time.Time
	Attrs map[string]any
}

// Span is one parsed span. End is the zero time when the span is unfinished.
type Span struct {
	TraceID, SpanID, ParentSpanID string
	Name                          string
	Start, End                    time.Time
	Attrs                         map[string]any
	Events                        []Event
	Status                        Status
}

// num holds a JSON number or numeric string; OTLP JSON writes 64 bit values as either.
// JSON null is the empty value, same as an absent field.
type num string

func (n *num) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*n = ""
		return nil
	}
	*n = num(strings.Trim(string(b), `"`))
	return nil
}

type wireValue struct {
	String *string  `json:"stringValue"`
	Bytes  *string  `json:"bytesValue"`
	Int    *num     `json:"intValue"`
	Double *float64 `json:"doubleValue"`
	Bool   *bool    `json:"boolValue"`
	Array  *struct {
		Values []wireValue `json:"values"`
	} `json:"arrayValue"`
	KV *struct {
		Values []wireAttr `json:"values"`
	} `json:"kvlistValue"`
}

type wireAttr struct {
	Key   string    `json:"key"`
	Value wireValue `json:"value"`
}

type wireSpan struct {
	TraceID      string     `json:"traceId"`
	SpanID       string     `json:"spanId"`
	ParentSpanID string     `json:"parentSpanId"`
	Name         string     `json:"name"`
	Start        num        `json:"startTimeUnixNano"`
	End          num        `json:"endTimeUnixNano"`
	Attrs        []wireAttr `json:"attributes"`
	Events       []struct {
		Name  string     `json:"name"`
		Time  num        `json:"timeUnixNano"`
		Attrs []wireAttr `json:"attributes"`
	} `json:"events"`
	Status struct {
		Code    num    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
}

type wireRequest struct {
	ResourceSpans []struct {
		ScopeSpans []struct {
			Spans []wireSpan `json:"spans"`
		} `json:"scopeSpans"`
	} `json:"resourceSpans"`
}

func (v wireValue) value() (any, error) {
	switch {
	case v.String != nil:
		return *v.String, nil
	case v.Bytes != nil:
		return *v.Bytes, nil
	case v.Int != nil:
		return strconv.ParseInt(string(*v.Int), 10, 64)
	case v.Double != nil:
		return *v.Double, nil
	case v.Bool != nil:
		return *v.Bool, nil
	case v.Array != nil:
		out := make([]any, 0, len(v.Array.Values))
		for _, e := range v.Array.Values {
			x, err := e.value()
			if err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		return out, nil
	case v.KV != nil:
		return attrMap(v.KV.Values)
	}
	return nil, nil
}

func attrMap(in []wireAttr) (map[string]any, error) {
	out := make(map[string]any, len(in))
	for _, a := range in {
		v, err := a.Value.value()
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", a.Key, err)
		}
		out[a.Key] = v
	}
	return out, nil
}

// nanosTime converts unix nanoseconds to UTC time. Empty or 0 is the zero time.
func nanosTime(n num) (time.Time, error) {
	if n == "" {
		return time.Time{}, nil
	}
	u, err := strconv.ParseUint(string(n), 10, 63)
	if err != nil {
		return time.Time{}, err
	}
	if u == 0 {
		return time.Time{}, nil
	}
	return time.Unix(0, int64(u)).UTC(), nil
}

func statusCode(n num) (StatusCode, error) {
	switch n {
	case "", "0", "STATUS_CODE_UNSET":
		return StatusUnset, nil
	case "1", "STATUS_CODE_OK":
		return StatusOK, nil
	case "2", "STATUS_CODE_ERROR":
		return StatusError, nil
	}
	return 0, fmt.Errorf("unknown status code %q", string(n))
}

func (w wireSpan) span() (Span, error) {
	s := Span{TraceID: w.TraceID, SpanID: w.SpanID, ParentSpanID: w.ParentSpanID, Name: w.Name}
	var err error
	if s.Start, err = nanosTime(w.Start); err != nil {
		return s, fmt.Errorf("start time: %w", err)
	}
	if s.End, err = nanosTime(w.End); err != nil {
		return s, fmt.Errorf("end time: %w", err)
	}
	if s.Attrs, err = attrMap(w.Attrs); err != nil {
		return s, err
	}
	for _, e := range w.Events {
		ev := Event{Name: e.Name}
		if ev.Time, err = nanosTime(e.Time); err != nil {
			return s, fmt.Errorf("event %q time: %w", e.Name, err)
		}
		if ev.Attrs, err = attrMap(e.Attrs); err != nil {
			return s, err
		}
		s.Events = append(s.Events, ev)
	}
	if s.Status.Code, err = statusCode(w.Status.Code); err != nil {
		return s, err
	}
	s.Status.Message = w.Status.Message
	return s, nil
}

// ParseOTLP parses raw OTLP JSON trace data as a file exporter writes it: one
// ExportTraceServiceRequest, or several concatenated or one per line (R-001). Spans come
// back in input order. Empty or malformed input is an error.
func ParseOTLP(data []byte) ([]Span, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var spans []Span
	n := 0
	for {
		var req wireRequest
		if err := dec.Decode(&req); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("otlp: %w", err)
		}
		n++
		for _, rs := range req.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				for _, w := range ss.Spans {
					s, err := w.span()
					if err != nil {
						return nil, fmt.Errorf("otlp: span %q: %w", w.Name, err)
					}
					spans = append(spans, s)
				}
			}
		}
	}
	if n == 0 {
		return nil, fmt.Errorf("otlp: no JSON input")
	}
	return spans, nil
}
