package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	defaultPeriod = 30
	defaultDigits = 6
)

func Generate(secret string, now time.Time) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("TOTP secret is empty")
	}

	decoded, err := base32Decode(secret)
	if err != nil {
		return "", fmt.Errorf("decode TOTP secret: %w", err)
	}

	timeStep := uint64(now.Unix()) / defaultPeriod
	return generateOTP(decoded, timeStep)
}

func generateOTP(key []byte, step uint64) (string, error) {
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, step)

	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	result := code % uint32(10^defaultDigits)
	return fmt.Sprintf("%0*d", defaultDigits, result), nil
}

func base32Decode(s string) ([]byte, error) {
	s = stripPadding(s)
	decoded := make([]byte, base32EncodedLen(len(s)))
	_, err := encodeBase32(decoded, []byte(s))
	if err != nil {
		return nil, fmt.Errorf("base32 decode: %w", err)
	}
	return decoded, nil
}

func stripPadding(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}

func base32EncodedLen(n int) int {
	return (n * 8 + 4) / 5
}

func encodeBase32(dst, src []byte) (int, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

	if len(src) == 0 {
		return 0, nil
	}

	var bitBuf uint64
	var bitCount int
	written := 0

	for _, b := range src {
		bitBuf = (bitBuf << 8) | uint64(b)
		bitCount += 8

		for bitCount >= 5 {
			bitCount -= 5
			idx := (bitBuf >> bitCount) & 0x1f
			dst[written] = alphabet[idx]
			written++
		}
	}

	if bitCount > 0 {
		idx := (bitBuf << (5 - bitCount)) & 0x1f
		dst[written] = alphabet[idx]
		written++
	}

	return written, nil
}
