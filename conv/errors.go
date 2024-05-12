package conv

import "errors"

var (
	ErrInvalidAddress     = errors.New("invalid address format, ok [protocol://][<NODE_ID>@]<IPv4>:<PORT>")
	ErrNilKey             = errors.New("key cannot be nil")
	ErrUnsupportedKeyType = errors.New("unsupported key type")
)
