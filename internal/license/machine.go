package license

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)



func GetMachineID() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getMacMachineID()
	case "linux":


		return "linux-placeholder-id", nil
	default:
		return "unknown-platform", nil
	}
}

func getMacMachineID() (string, error) {
	cmd := exec.Command("ioreg", "-d2", "-c", "IOPlatformExpertDevice")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}

	output := out.String()

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				id := strings.TrimSpace(parts[1])
				id = strings.Trim(id, "\"")

				hash := sha256.Sum256([]byte(id))
				return hex.EncodeToString(hash[:]), nil
			}
		}
	}
	return "", fmt.Errorf("UUID not found in ioreg output")
}
