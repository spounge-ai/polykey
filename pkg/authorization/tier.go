package authorization

import (
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// ValidateTierForProfile checks if a client of a certain tier can use the specified storage profile.
// It includes input validation and returns a structured error.
func ValidateTierForProfile(tier domain.KeyTier, profile pk.StorageProfile) error {
	// Input validation
	if tier == "" {
		tier = domain.TierUnknown // Default to unknown/free tier
	}

	switch tier {
	case domain.TierEnterprise, domain.TierPro:
		// These tiers can use any profile.
		return nil
	case domain.TierFree, domain.TierUnknown:
		if profile == pk.StorageProfile_STORAGE_PROFILE_HARDENED {
			return fmt.Errorf("tier '%s' is not permitted to use the hardened storage profile", tier)
		}
		return nil
	default:
		// This case handles any unexpected tier values that might be introduced.
		// We also check for hardened storage here as a safeguard.
		if profile == pk.StorageProfile_STORAGE_PROFILE_HARDENED {
			return fmt.Errorf("tier '%s' is not permitted to use the hardened storage profile", tier)
		}
		return nil
	}
}
