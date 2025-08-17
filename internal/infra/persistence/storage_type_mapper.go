package persistence

import (
	consts "github.com/spounge-ai/polykey/internal/constants"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Pre-compiled storage type mappings for better performance
var storageTypeMap = map[pk.StorageProfile]string{
	pk.StorageProfile_STORAGE_PROFILE_STANDARD: consts.StorageTypeStandard,
	pk.StorageProfile_STORAGE_PROFILE_HARDENED: consts.StorageTypeHardened,
}

// getStorageTypeOptimized uses pre-compiled map for better performance
func getStorageTypeOptimized(storageProfile pk.StorageProfile) string {
	if storageType, ok := storageTypeMap[storageProfile]; ok {
		return storageType
	}
	return consts.StorageTypeUnknown
}
