package tools

import (
	"crypto/sha256"
	"encoding/hex"
)

func GenLayerID(parentID, ublob string) string {
	data := parentID + "-" + ublob
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
