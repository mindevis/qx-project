package servers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateRconPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate rcon password: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func rconPortFor(gamePort int) int {
	if gamePort <= 0 {
		gamePort = 25565
	}
	if gamePort <= 65535-10000 {
		return gamePort + 10000
	}
	if gamePort > 1 {
		return gamePort - 1
	}
	return 25575
}
