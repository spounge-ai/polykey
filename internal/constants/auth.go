package constants

const (
	MethodGetKey            = "GetKey"
	MethodCreateKey         = "CreateKey"
	MethodListKeys          = "ListKeys"
	MethodRotateKey         = "RotateKey"
	MethodRevokeKey         = "RevokeKey"
	MethodUpdateKeyMetadata = "UpdateKeyMetadata"
	MethodGetKeyMetadata    = "GetKeyMetadata"
)

const (
	AuthKeysRead   = "keys:read"
	AuthKeysCreate = "keys:create"
	AuthKeysList   = "keys:list"
	AuthKeysRotate = "keys:rotate"
	AuthKeysRevoke = "keys:revoke"
	AuthKeysUpdate = "keys:update"
)
 
var MethodScopes = map[string]string{
	MethodGetKey:            AuthKeysRead,
	MethodCreateKey:         AuthKeysCreate,
	MethodListKeys:          AuthKeysList,
	MethodRotateKey:         AuthKeysRotate,
	MethodRevokeKey:         AuthKeysRevoke,
	MethodUpdateKeyMetadata: AuthKeysUpdate,
	MethodGetKeyMetadata:    AuthKeysRead,
}
