package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

func wrapConnectError(err error) error {
	if err == nil {
		return nil
	}
	if isDBUnreachable(err) {
		return fmt.Errorf("%w", err)
	}
	return fmt.Errorf("failed to connect to database: %w", err)
}

func isDBUnreachable(err error) bool {
	for err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return true
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return true
		}
		msg := strings.ToLower(err.Error())
		for _, sub := range []string{
			"connection refused",
			"connectex",
			"actively refused",
			"no connection could be made",
			"no such host",
			"network is unreachable",
			"connection reset",
			"i/o timeout",
			"dial tcp",
		} {
			if strings.Contains(msg, sub) {
				return true
			}
		}
		err = errors.Unwrap(err)
	}
	return false
}
