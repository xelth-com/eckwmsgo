package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"os"
	"time"
)

// Base32Chars is the alphabet for human-readable warehouse locations and encrypted URLs
const Base32Chars = "0123456789ABCDEFGHJKLMNPQRTUVWXY"

var base32BackHash = make(map[byte]int)

func init() {
	for i, c := range []byte(Base32Chars) {
		base32BackHash[c] = i
	}
}

// ToBase32Char converts a number to a base32 character
func ToBase32Char(num int) string {
	if num < 0 || num >= len(Base32Chars) {
		return "?"
	}
	return string(Base32Chars[num])
}

// timeIvBase32 generates a time-based IV using Base32 encoding
func timeIvBase32(ivLength int) []byte {
	if ivLength < 3 {
		ivLength = 3
	}
	tempTime := time.Now().UnixMilli() / 7777777
	buf := make([]byte, ivLength)

	// Fill with random bytes first
	rand.Read(buf[:ivLength-3])

	// Convert all bytes to Base32
	for i := 0; i < ivLength; i++ {
		if i >= ivLength-3 {
			buf[i] = Base32Chars[(tempTime>>(5*(ivLength-1-i)))&31]
		} else {
			buf[i] = Base32Chars[buf[i]&31]
		}
	}
	return buf
}

// toBase32 converts buffer to Base32 encoded buffer
func toBase32(bufIn []byte) []byte {
	iterat := len(bufIn) / 5
	bufOut := make([]byte, iterat*8)

	for i := 0; i < iterat; i++ {
		bufOut[i*8+0] = Base32Chars[bufIn[i*5+0]>>3]
		bufOut[i*8+1] = Base32Chars[((bufIn[i*5+0]<<2)+(bufIn[i*5+1]>>6))&31]
		bufOut[i*8+2] = Base32Chars[(bufIn[i*5+1]>>1)&31]
		bufOut[i*8+3] = Base32Chars[((bufIn[i*5+1]<<4)+(bufIn[i*5+2]>>4))&31]
		bufOut[i*8+4] = Base32Chars[((bufIn[i*5+2]<<1)+(bufIn[i*5+3]>>7))&31]
		bufOut[i*8+5] = Base32Chars[(bufIn[i*5+3]>>2)&31]
		bufOut[i*8+6] = Base32Chars[((bufIn[i*5+3]<<3)+(bufIn[i*5+4]>>5))&31]
		bufOut[i*8+7] = Base32Chars[(bufIn[i*5+4])&31]
	}

	return bufOut
}

// fromBase32 converts Base32 encoded buffer back to original format
func fromBase32(bufIn []byte) []byte {
	iterat := len(bufIn) / 8
	bufOut := make([]byte, iterat*5)

	for i := 0; i < iterat; i++ {
		bufOut[i*5+0] = byte((base32BackHash[bufIn[i*8+0]] << 3) + (base32BackHash[bufIn[i*8+1]] >> 2))
		bufOut[i*5+1] = byte((base32BackHash[bufIn[i*8+1]] << 6) + (base32BackHash[bufIn[i*8+2]] << 1) + (base32BackHash[bufIn[i*8+3]] >> 4))
		bufOut[i*5+2] = byte((base32BackHash[bufIn[i*8+3]] << 4) + (base32BackHash[bufIn[i*8+4]] >> 1))
		bufOut[i*5+3] = byte((base32BackHash[bufIn[i*8+4]] << 7) + (base32BackHash[bufIn[i*8+5]] << 2) + (base32BackHash[bufIn[i*8+6]] >> 3))
		bufOut[i*5+4] = byte((base32BackHash[bufIn[i*8+6]] << 5) + (base32BackHash[bufIn[i*8+7]]))
	}

	return bufOut
}

// EckURLEncrypt encrypts a string and formats it for use in a URL (AES-192-GCM)
func EckURLEncrypt(plaintext string) (string, error) {
	encKeyHex := os.Getenv("ENC_KEY")
	if encKeyHex == "" {
		return "", errors.New("ENC_KEY environment variable not set")
	}

	key, err := hex.DecodeString(encKeyHex)
	if err != nil {
		return "", errors.New("invalid ENC_KEY format")
	}

	// AES-192 requires 24-byte key
	if len(key) != 24 {
		return "", errors.New("ENC_KEY must be 24 bytes (48 hex chars) for AES-192")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return "", err
	}

	// Generate time-based IV (9 bytes Base32)
	betIv := timeIvBase32(9)

	// Create full IV by concatenating (matching Node.js behavior)
	iv := make([]byte, 16)
	copy(iv[:9], betIv)
	copy(iv[9:], betIv[:7])

	// Encrypt
	ciphertext := aesGCM.Seal(nil, iv, []byte(plaintext), nil)

	// ciphertext includes the auth tag at the end (16 bytes for GCM)
	// Encode to Base32
	base32Encoded := toBase32(ciphertext)

	// Combine Base32 data and Base32 IV
	result := append(base32Encoded, betIv...)
	return string(result), nil
}

// EckURLDecrypt decrypts a URL-formatted encrypted string
func EckURLDecrypt(encryptedURL string) (string, error) {
	instanceSuffix := os.Getenv("INSTANCE_SUFFIX")
	if instanceSuffix == "" {
		return "", errors.New("INSTANCE_SUFFIX environment variable not set")
	}

	// Validate format: ECKn.COM/ prefix (9 chars) + data (56 chars) + IV (9 chars) + suffix (2 chars) = 76
	if len(encryptedURL) != 76 {
		return "", errors.New("invalid encrypted URL length")
	}

	prefix := encryptedURL[0:9]
	if prefix != "ECK1.COM/" && prefix != "ECK2.COM/" && prefix != "ECK3.COM/" {
		return "", errors.New("invalid URL prefix")
	}

	suffix := encryptedURL[74:76]
	if suffix != instanceSuffix {
		return "", errors.New("invalid instance suffix")
	}

	encKeyHex := os.Getenv("ENC_KEY")
	if encKeyHex == "" {
		return "", errors.New("ENC_KEY environment variable not set")
	}

	key, err := hex.DecodeString(encKeyHex)
	if err != nil {
		return "", errors.New("invalid ENC_KEY format")
	}

	if len(key) != 24 {
		return "", errors.New("ENC_KEY must be 24 bytes for AES-192")
	}

	// Extract Base32 IV (chars 65-74) and Base32 data (chars 9-65)
	betIv := []byte(encryptedURL[65:74])
	base32Data := []byte(encryptedURL[9:65])

	// Decode from Base32
	decodedData := fromBase32(base32Data)

	// Reconstruct IV
	iv := make([]byte, 16)
	copy(iv[:9], betIv)
	copy(iv[9:], betIv[:7])

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return "", err
	}

	// Decrypt (decodedData includes ciphertext + auth tag)
	plaintext, err := aesGCM.Open(nil, iv, decodedData, nil)
	if err != nil {
		return "", errors.New("decryption failed: invalid auth tag or corrupted data")
	}

	return string(plaintext), nil
}

// EckCRC generates a 2-character CRC check value using Base32
func EckCRC(value int) string {
	temp := crc32.ChecksumIEEE([]byte(string(rune(value)))) & 1023
	return string(Base32Chars[temp>>5]) + string(Base32Chars[temp&31])
}

// EckCRCFromString generates a 2-character CRC check value from string
func EckCRCFromString(value string) string {
	temp := crc32.ChecksumIEEE([]byte(value)) & 1023
	return string(Base32Chars[temp>>5]) + string(Base32Chars[temp&31])
}
