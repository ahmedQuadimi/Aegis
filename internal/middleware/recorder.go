package middleware

import "net/http"

type StatusRecorder struct {
	http.ResponseWriter
	Status  int
	Written int64
}

func (r *StatusRecorder) WriteHeader(statusCode int) {
	r.Status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *StatusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.Written += int64(n)
	return n, err
}
