package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func decodePyrogramSessionString(encodedString string) (string, error) {
	const authKeySize = 256
	s := strings.TrimRight(encodedString, "=")
	packedData, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(s)
	if err != nil {
		packedData, err = base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			return "", fmt.Errorf("decode session: %w", err)
		}
	}
	if len(packedData) < 1+4+1+authKeySize {
		return "", fmt.Errorf("unexpected data length: got %d", len(packedData))
	}
	dcID := packedData[0]
	appID := binary.BigEndian.Uint32(packedData[1:5])
	testMode := packedData[5] != 0
	key := packedData[6 : 6+authKeySize]
	sess := &telegram.Session{
		Hostname: telegram.ResolveDC(int(dcID), testMode, false),
		AppID:    int32(appID),
		Key:      key,
	}
	return sess.Encode(), nil
}
