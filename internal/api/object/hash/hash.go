package hash

import (
	"encoding/hex"
	"errors"
	"fmt"
)

func IsObjectHash(objectHash string) bool {
	_, err := GetBinaryHash(objectHash)
	return err == nil
}

func GetBinaryHash(objectHash string) ([]byte, error) {
	if objectHash == "" {
		err := errors.New("objectHash must not be empty")
		return nil, err
	}

	// we're using sha256 and therefore the provided hash should consist of 32 bytes
	if hex.DecodedLen(len(objectHash)) != 32 {
		err := errors.New("objectHash has unexpected size")
		return nil, err
	}

	binaryHash, err := hex.DecodeString(objectHash)
	if err != nil {
		return nil, fmt.Errorf("hex-decode objectHash: %w", err)
	}

	return binaryHash, nil // the objectHash can be parsed and contains a valid representation of 32 bits
}
