package utils

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// Base32 Alphabet (0-9, A-Z excluding confusable chars if needed, standard Crockford or RFC)
// Using standard alphanumeric for max density
const SmartBase32Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Epoch for Smart Labels (Jan 1, 2025)
var SmartLabelEpoch = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// ==========================================
// 1. SMART ITEM ('i')
// Format: i [SplitChar] [Serial...] [EAN...]
// SplitChar: Base32 digit representing the length of the SUFFIX (EAN/ID).
// Example: Split 'D' (13) means last 13 chars are EAN, rest is Serial.
// ==========================================

type SmartItemData struct {
	Serial string // The unique part (Internal Ref)
	RefID  string // The identification part (EAN, UPC, etc.)
}

func DecodeSmartItem(code string) (*SmartItemData, error) {
	if len(code) < 3 || !strings.HasPrefix(code, "i") {
		return nil, errors.New("invalid item code")
	}

	code = strings.ToUpper(code)

	// 2nd char is Split Length (Base32)
	splitChar := string(code[1])
	suffixLen := base32ToInt(splitChar)

	dataPart := code[2:]
	if len(dataPart) < suffixLen {
		return nil, errors.New("code too short for specified split length")
	}

	// Split
	splitIdx := len(dataPart) - suffixLen
	serial := dataPart[:splitIdx]
	refID := dataPart[splitIdx:]

	return &SmartItemData{
		Serial: serial,
		RefID:  refID,
	}, nil
}

func GenerateSmartItem(serial, refID string) string {
	suffixLen := len(refID)
	// Safety check: max suffix length 35 (Z in Base36)
	if suffixLen > 35 {
		suffixLen = 35 // Truncate or error in real app
	}

	splitChar := intToBase32(suffixLen, 1)
	return fmt.Sprintf("i%s%s%s", splitChar, serial, refID)
}

// ==========================================
// 2. SMART BOX ('b')
// Format: b LL WW HH MMM T SSSSSSSS
// ==========================================

type SmartBoxData struct {
	Length int     // cm (0-1023)
	Width  int     // cm (0-1023)
	Height int     // cm (0-1023)
	Weight float64 // kg (Tiered precision)
	Type   string  // Package Type char (P=Pallet, B=Box, etc)
	Serial uint64  // Numeric ID
}

func DecodeSmartBox(code string) (*SmartBoxData, error) {
	if len(code) != 19 || !strings.HasPrefix(code, "b") {
		return nil, errors.New("invalid box code")
	}
	code = strings.ToUpper(code)

	l := base32ToInt(code[1:3])
	w := base32ToInt(code[3:5])
	h := base32ToInt(code[5:7])
	mVal := base32ToInt(code[7:10])
	t := string(code[10])
	serial := base32ToInt(code[11:19])

	return &SmartBoxData{
		Length: l, Width: w, Height: h,
		Weight: decodeWeight(mVal),
		Type:   t,
		Serial: uint64(serial),
	}, nil
}

func GenerateSmartBox(data SmartBoxData) (string, error) {
	if data.Length > 1023 || data.Width > 1023 || data.Height > 1023 {
		return "", errors.New("dimensions too large")
	}
	mVal, err := encodeWeight(data.Weight)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("b%s%s%s%s%s%s",
		intToBase32(data.Length, 2),
		intToBase32(data.Width, 2),
		intToBase32(data.Height, 2),
		intToBase32(mVal, 3),
		strings.ToUpper(data.Type[:1]),
		intToBase32(int(data.Serial), 8),
	), nil
}

// ==========================================
// 3. SMART LABEL ('l')
// Format: l DDD T SSSSSSSSSSSSSS
// DDD: Days since 2025-01-01
// T: Type (Action, Status, User, etc)
// S...: Payload (14 chars)
// ==========================================

type SmartLabelData struct {
	Date    time.Time
	Type    string
	Payload string
}

func DecodeSmartLabel(code string) (*SmartLabelData, error) {
	if len(code) != 19 || !strings.HasPrefix(code, "l") {
		return nil, errors.New("invalid label code")
	}
	code = strings.ToUpper(code)

	days := base32ToInt(code[1:4])
	date := SmartLabelEpoch.AddDate(0, 0, days)
	t := string(code[4])
	payload := code[5:]

	return &SmartLabelData{
		Date:    date,
		Type:    t,
		Payload: payload,
	}, nil
}

func GenerateSmartLabel(data SmartLabelData) (string, error) {
	if data.Date.Before(SmartLabelEpoch) {
		return "", errors.New("date too old")
	}
	days := int(data.Date.Sub(SmartLabelEpoch).Hours() / 24)
	if days > 46655 { // 36^3 - 1
		return "", errors.New("date too far future")
	}

	return fmt.Sprintf("l%s%s%s",
		intToBase32(days, 3),
		strings.ToUpper(data.Type[:1]),
		padRight(data.Payload, 14, "0"),
	), nil
}

// ==========================================
// HELPERS & LOGIC
// ==========================================

// Weight Tiers:
// 1: 0-20kg (10g step) -> 0..2000
// 2: 20-1000kg (100g step) -> 2000..11800
// 3: 1000-30000kg (1kg step) -> 11800..40800
const (
	Tier1Limit = 20.0
	Tier1Step  = 0.01
	Tier1Max   = 2000
	Tier2Limit = 1000.0
	Tier2Step  = 0.1
	Tier2Max   = 11800
	Tier3Limit = 30000.0
	Tier3Step  = 1.0
)

func encodeWeight(kg float64) (int, error) {
	if kg <= Tier1Limit {
		return int(math.Round(kg / Tier1Step)), nil
	}
	if kg <= Tier2Limit {
		return Tier1Max + int(math.Round((kg-Tier1Limit)/Tier2Step)), nil
	}
	if kg <= Tier3Limit {
		return Tier2Max + int(math.Round((kg-Tier2Limit)/Tier3Step)), nil
	}
	return 0, errors.New("weight too heavy")
}

func decodeWeight(val int) float64 {
	if val <= Tier1Max {
		return float64(val) * Tier1Step
	}
	if val <= Tier2Max {
		return Tier1Limit + float64(val-Tier1Max)*Tier2Step
	}
	return Tier2Limit + float64(val-Tier2Max)*Tier3Step
}

func base32ToInt(chunk string) int {
	val := 0
	for _, char := range chunk {
		idx := strings.IndexRune(SmartBase32Chars, char)
		if idx == -1 {
			return 0
		}
		val = val*len(SmartBase32Chars) + idx
	}
	return val
}

func intToBase32(num, width int) string {
	base := len(SmartBase32Chars)
	res := ""
	for i := 0; i < width; i++ {
		rem := num % base
		res = string(SmartBase32Chars[rem]) + res
		num = num / base
	}
	return res
}

func padRight(s string, l int, pad string) string {
	if len(s) >= l {
		return s[:l]
	}
	return s + strings.Repeat(pad, l-len(s))
}
