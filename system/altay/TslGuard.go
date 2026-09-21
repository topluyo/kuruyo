package main


import (
	"log/slog"
	"regexp"
	"strings"
)


type tlsErrorLogger struct{}

var tlsHandshakeErrorRegex = regexp.MustCompile(
	`TLS handshake error from ([^:]+):[0-9]+:`,
)

func (tlsErrorLogger) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))

	if strings.Contains(msg, "TLS handshake error") {
		matches := tlsHandshakeErrorRegex.FindStringSubmatch(msg)

		if len(matches) >= 2 {
      ip := matches[1]
      go write(client.Request("DROP,"+ip+",90"))

			slog.Error("TLS handshake failed",
				"ip", ip,
				"error", msg,
			)
		} else {
			slog.Error("TLS handshake failed",
				"error", msg,
			)
		}

		return len(p), nil
	}

	// Diğer http.Server hatalarını da kaybetme.
	slog.Error("http server error", "error", msg)

	return len(p), nil
}