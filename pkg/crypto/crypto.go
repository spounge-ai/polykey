package crypto

import (
	"fmt"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

var (
	ErrInvalidKeyType = fmt.Errorf("invalid key type")
)

func GetCryptoDetails(keyType pk.KeyType) (int, string, error) {
	switch keyType {
	case pk.KeyType_KEY_TYPE_AES_256:
		return 32, "AES-256-GCM", nil
	default:
		return 0, "", fmt.Errorf("%w: %s", ErrInvalidKeyType, keyType.String())
	}
}
