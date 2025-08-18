package authorization

import (
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	cmn "github.com/spounge-ai/spounge-proto/gen/go/common/v2"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// FromProtoTier converts a protobuf ClientTier enum to a domain KeyTier string.
func FromProtoTier(tier cmn.ClientTier) domain.KeyTier {
	switch tier {
	case cmn.ClientTier_CLIENT_TIER_FREE:
		return domain.TierFree
	case cmn.ClientTier_CLIENT_TIER_PRO:
		return domain.TierPro
	case cmn.ClientTier_CLIENT_TIER_ENTERPRISE:
		return domain.TierEnterprise
	default:
		return domain.TierUnknown
	}
}

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
