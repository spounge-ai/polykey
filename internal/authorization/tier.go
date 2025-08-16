package authorization

import (
	"fmt"

	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// ValidateStorageProfileForTier checks if a client of a certain tier can use the specified storage profile.
func ValidateStorageProfileForTier(tier domain.KeyTier, profile pk.StorageProfile) error {
	switch tier {
	case domain.TierEnterprise:
		return nil // Enterprise can use any profile.
	case domain.TierPro:
		return nil // Pro can use any profile.
	case domain.TierFree:
		if profile == pk.StorageProfile_STORAGE_PROFILE_HARDENED {
			return fmt.Errorf("free tier clients cannot use the hardened storage profile")
		}
		return nil
	default: // Unkown tier defaults to free.
		if profile == pk.StorageProfile_STORAGE_PROFILE_HARDENED {
			return fmt.Errorf("clients with no tier cannot use the hardened storage profile")
		}
		return nil
	}
}
