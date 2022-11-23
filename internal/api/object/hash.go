package object

import "encoding/hex"

func IsObjectHash(objectHash string) bool {
	if objectHash == "" {
		return false
	}

	// we're using sha256 and therefore the provided hash should consist of 32 bytes
	if hex.DecodedLen(len(objectHash)) != 32 {
		return false
	}

	_, err := hex.DecodeString(objectHash)
	return err == nil // the objectHash can be parsed and contains a valid representation of 32 bits
}
