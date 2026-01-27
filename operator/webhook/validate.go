package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-version"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"k8s.io/apimachinery/pkg/runtime"
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
func GetAvailableZones(ctx context.Context) ([]string, error) {
	// we do a direct http request here because the sdk validates credentials at client creation
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api-ch-gva-2.exoscale.com/v2/zone", nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list zones request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list zones failed with status %d", resp.StatusCode)
	}

	var zonesResp zonesResponse
	if err := json.NewDecoder(resp.Body).Decode(&zonesResp); err != nil {
		return nil, fmt.Errorf("decode zones response failed: %w", err)
	}

	zones := make([]string, 0, len(zonesResp.Zones))
	for _, zone := range zonesResp.Zones {
		zones = append(zones, zone.Name)
	}

	return zones, nil
}

// ValidateZoneExists fetches available zones and validates that the requested zone exists.
func ValidateZoneExists(ctx context.Context, requestedZone string) error {
	availableZones, err := GetAvailableZones(ctx)
	if err != nil {
		return err
	}
	return ValidateZone(requestedZone, availableZones)
}
