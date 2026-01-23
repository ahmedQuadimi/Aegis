package lb

import (
	"net"
	"net/http"
	"time"
)

func RunHealthCheck(b *Backend) {
	// If method is passive, we don't start a background ticker
	if b.Config.Method == "passive" {
		return
	}

	ticker := time.NewTicker(time.Duration(b.Config.Interval) * time.Millisecond)
	client := &http.Client{
		Timeout: time.Duration(b.Config.Timeout) * time.Millisecond,
	}

	for range ticker.C {
		var isHealthy bool

		switch b.Config.Method {
		case "http":
			// We check for 200 OK on the specific path (e.g., /health)
			resp, err := client.Get(b.Addr + b.Config.Path)
			isHealthy = (err == nil && resp.StatusCode == http.StatusOK)
			if resp != nil {
				resp.Body.Close()
			}

		case "tcp":
			// Fast L4 check
			host, _ := b.GetHost()
			conn, err := net.DialTimeout("tcp", host, time.Duration(b.Config.Timeout)*time.Millisecond)
			isHealthy = (err == nil)
			if conn != nil {
				conn.Close()
			}
		}

		b.UpdateStatus(isHealthy)
	}
}
