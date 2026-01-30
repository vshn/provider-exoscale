package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func ValidateRawExtension(raw runtime.RawExtension) error {
	m, err := mapper.ToMap(raw)
	if err != nil {
		return fmt.Errorf("mapper.ToMap(%q): %w", raw, err)
	}
	for k, v := range m {
		switch v.(type) {
		case string:
		case int64:
		case float64:
		case bool:
			continue
		default:
			return fmt.Errorf("validate: value of key %q is not a supported type (only strings, boolean and numbers): %v", k, v)
		}
	}
	return nil
}

func ValidateUpdateVersion(oldObs, oldDes, newDes string) error {
	oldObserved, err := version.NewVersion(oldObs)
	if err != nil {
		return fmt.Errorf("set old status version '%s' failed: %w", oldObs, err)
	}
	oldDesired, err := version.NewVersion(oldDes)
	if err != nil {
		return fmt.Errorf("set old desired version '%s' failed: %w", oldDes, err)
	}
	newDesired, err := version.NewVersion(newDes)
	if err != nil {
		return fmt.Errorf("set new desired version '%s' failed: %w", newDes, err)
	}

	c, err := version.NewConstraint(fmt.Sprintf("<= %s", oldObserved.String()))
	if err != nil {
		return fmt.Errorf("set version constraint failed: %w", err)
	}

	if newDesired.GreaterThan(oldDesired) && !c.Check(newDesired) {
		// update "14" -> "15.1" not allowed if observed version is < "15.1"
		return fmt.Errorf("field is immutable after creation: %s (old), %s (changed)", oldDesired, newDesired)
	}
	// we only allow version change if it matches the observed version
	return nil
}

func ValidateVersions(wanted string, admittedVersions []string) error {
	if wanted == "" {
		return fmt.Errorf("version must be provided")
	}
	if !contains(admittedVersions, wanted) {
		return fmt.Errorf("version not valid")
	}
	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// ValidateZone validates that the requested zone exists in the list of available zones.
func ValidateZone(requestedZone string, availableZones []string) error {
	if requestedZone == "" {
		return fmt.Errorf("zone must be provided")
	}
	if !contains(availableZones, requestedZone) {
		return fmt.Errorf("zone %q is not valid, available zones: %v", requestedZone, availableZones)
	}
	return nil
}

// zonesResponse represents the response from the Exoscale v2 /zone endpoint
type zonesResponse struct {
	Zones []struct {
		Name string `json:"name"`
	} `json:"zones"`
}

// GetAvailableZones fetches the list of available zones from the Exoscale API.
// It tries multiple zone endpoints as fallback in case one is unavailable.
func GetAvailableZones(ctx context.Context) ([]string, error) {
	// Default endpoints to try, ordered by preference
	endpoints := []string{
		"https://api-ch-gva-2.exoscale.com/v2/zone",
		"https://api-de-fra-1.exoscale.com/v2/zone",
		"https://api-ch-dk-2.exoscale.com/v2/zone",
		"https://api-at-vie-1.exoscale.com/v2/zone",
	}

	// Allow overriding via environment variable
	if envEndpoints := os.Getenv("EXOSCALE_ZONES_API_ENDPOINTS"); envEndpoints != "" {
		endpoints = strings.Split(envEndpoints, ",")
		for i, e := range endpoints {
			endpoints[i] = strings.TrimSpace(e)
		}
	}

	var lastErr error
	// we do a direct http request here because the sdk validates credentials at client creation
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, endpoint := range endpoints {
		zones, err := tryFetchZones(ctx, client, endpoint)
		if err == nil {
			return zones, nil
		}
		lastErr = err
	}

	// If all endpoints failed, return the last error
	return nil, fmt.Errorf("failed to fetch zones from all endpoints: %w", lastErr)
}

func tryFetchZones(ctx context.Context, client *http.Client, endpoint string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list zones request to %s failed: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list zones from %s failed with status %d, body: %s", endpoint, resp.StatusCode, string(body))
	}

	var zonesResp zonesResponse
	if err := json.NewDecoder(resp.Body).Decode(&zonesResp); err != nil {
		return nil, fmt.Errorf("decode zones response from %s failed: %w", endpoint, err)
	}

	zones := make([]string, 0, len(zonesResp.Zones))
	for _, zone := range zonesResp.Zones {
		zones = append(zones, zone.Name)
	}

	return zones, nil
}

// ValidateZoneExists fetches available zones and validates that the requested zone exists.
func ValidateZoneExists(ctx context.Context, requestedZone string) (admission.Warnings, error) {
	availableZones, err := GetAvailableZones(ctx)
	if err != nil {
		// soft fail to prevent blocking operations when API is unavailable
		warning := fmt.Sprintf("Unable to validate zone %q against Exoscale API (API may be temporarily unavailable): %v. Proceeding without validation.", requestedZone, err)
		return admission.Warnings{warning}, nil
	}
	return nil, ValidateZone(requestedZone, availableZones)
}
