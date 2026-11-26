package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
)

// NewOTP creates an unbiased six digit code using rejection sampling.
func NewOTP() (string, error) {
	var b [4]byte
	for {
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		n := binary.BigEndian.Uint32(b[:])
		if n < (1<<32)-((1<<32)%1_000_000) {
			return fmt.Sprintf("%06d", n%1_000_000), nil
		}
	}
}

// OTPDigest binds a code to its challenge and purpose using the dedicated key.
func OTPDigest(key []byte, challengeID, purpose, code string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(challengeID))
	h.Write([]byte{0})
	h.Write([]byte(purpose))
	h.Write([]byte{0})
	h.Write([]byte(code))
	return h.Sum(nil)
}

// VerifyOTPDigest performs a constant time digest comparison.
func VerifyOTPDigest(key []byte, challengeID, purpose, code string, digest []byte) bool {
	got := OTPDigest(key, challengeID, purpose, code)
	return len(digest) == len(got) && subtle.ConstantTimeCompare(got, digest) == 1
}
