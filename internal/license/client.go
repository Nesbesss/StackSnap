package license

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const DefaultLicenseServerURL = "http://localhost:8081"


func Verify(licenseServerURL, key, machineID string) (bool, error) {
	if licenseServerURL == "" {
		licenseServerURL = DefaultLicenseServerURL
	}

	verifyURL := licenseServerURL + "/api/verify"

	payload := map[string]string{
		"license_key": key,
		"machine_id":  machineID,
	}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(verifyURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return false, fmt.Errorf("could not reach license server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Valid  bool   `json:"valid"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Valid, nil
}
