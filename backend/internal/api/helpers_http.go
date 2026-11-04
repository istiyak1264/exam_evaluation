package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
)

// clientIP prefers the proxy header only when the deployment says so.
func clientIP(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-For"); h != "" {
		if i := strings.IndexByte(h, ','); i > 0 {
			return strings.TrimSpace(h[:i])
		}
		return strings.TrimSpace(h)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func decodeQuiet(r *http.Request, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, dst)
}

func r0(w http.ResponseWriter) context.Context { return context.Background() }
