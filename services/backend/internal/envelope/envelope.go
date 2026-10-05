// Package envelope is the response shape shared by every backend blueprint:
// { success, code, message, data, meta, trace_id }.
package envelope

type Envelope struct {
	Success bool           `json:"success"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Data    any            `json:"data"`
	Meta    map[string]any `json:"meta"`
	TraceID string         `json:"trace_id"`
}

func OK(data any, traceID string) Envelope {
	return Envelope{Success: true, Code: "OK", Message: "Request completed successfully", Data: data, Meta: map[string]any{}, TraceID: traceID}
}

func Failure(code, message, traceID string, meta map[string]any) Envelope {
	if meta == nil {
		meta = map[string]any{}
	}
	return Envelope{Success: false, Code: code, Message: message, Data: nil, Meta: meta, TraceID: traceID}
}
