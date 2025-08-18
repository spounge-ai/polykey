package authorization

import (
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// ValidateTierForProfile checks if a client of a certain tier can use the specified storage profile.
// It includes input validation and returns a structured error.
func ValidateTierForProfile(tier domain.KeyTier, profile pk.StorageProfile) error {
	// A user's tier determines the BEST profile they can use.
	// They are, however, allowed to use any profile at or below their level.
	maxProfile := GetStorageProfileForTier(tier)

	// This logic assumes that HARDENED > STANDARD.
	if profile > maxProfile {
		return fmt.Errorf("tier '%s' is not permitted to use the '%s' storage profile", tier, profile.String())
	}

	return nil
}

// GetStorageProfileForTier determines the appropriate storage profile based on the client's tier.
func GetStorageProfileForTier(tier domain.KeyTier) pk.StorageProfile {
	if tier == domain.TierPro || tier == domain.TierEnterprise {
		return pk.StorageProfile_STORAGE_PROFILE_HARDENED
	}
	return pk.StorageProfile_STORAGE_PROFILE_STANDARD
}
