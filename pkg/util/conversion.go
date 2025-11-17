package util

import (
	"encoding/hex"
	"fmt"
	"strconv"
)

func HexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

func PlmnIdToString(mcc, mnc string) string {
	return mcc + mnc
}

func StringToPlmnId(plmnId string) (mcc, mnc string, err error) {
	if len(plmnId) < 5 || len(plmnId) > 6 {
		return "", "", fmt.Errorf("invalid PLMN ID length")
	}

	mcc = plmnId[0:3]
	mnc = plmnId[3:]
	return mcc, mnc, nil
}

func SupiToImsi(supi string) (string, error) {
	if len(supi) < 6 {
		return "", fmt.Errorf("invalid SUPI format")
	}

	if supi[0:5] != "imsi-" {
		return "", fmt.Errorf("SUPI does not contain IMSI")
	}

	return supi[5:], nil
}

func ImsiToSupi(imsi string) string {
	return "imsi-" + imsi
}

func ParseAmfId(amfIdHex string) (regionId, setId, pointer string, err error) {
	amfIdInt, err := strconv.ParseUint(amfIdHex, 16, 24)
	if err != nil {
		return "", "", "", err
	}

	region := (amfIdInt >> 16) & 0xFF
	set := (amfIdInt >> 6) & 0x3FF
	ptr := amfIdInt & 0x3F

	regionId = fmt.Sprintf("%02x", region)
	setId = fmt.Sprintf("%03x", set)
	pointer = fmt.Sprintf("%02x", ptr)

	return regionId, setId, pointer, nil
}

func ConstructAmfId(regionId, setId, pointer string) (string, error) {
	region, err := strconv.ParseUint(regionId, 16, 8)
	if err != nil {
		return "", err
	}

	set, err := strconv.ParseUint(setId, 16, 10)
	if err != nil {
		return "", err
	}

	ptr, err := strconv.ParseUint(pointer, 16, 6)
	if err != nil {
		return "", err
	}

	amfId := (region << 16) | (set << 6) | ptr
	return fmt.Sprintf("%06x", amfId), nil
}

func ParseTac(tacHex string) (uint32, error) {
	tac, err := strconv.ParseUint(tacHex, 16, 24)
	if err != nil {
		return 0, err
	}
	return uint32(tac), nil
}

func TacToHex(tac uint32) string {
	return fmt.Sprintf("%06x", tac)
}
