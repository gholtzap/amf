package nas

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/gavin/amf/internal/context"
)

const (
	AlgorithmNEA0 = 0x00
	AlgorithmNEA1 = 0x01
	AlgorithmNEA2 = 0x02
	AlgorithmNEA3 = 0x03

	AlgorithmNIA0 = 0x00
	AlgorithmNIA1 = 0x01
	AlgorithmNIA2 = 0x02
	AlgorithmNIA3 = 0x03
)

func EncodeSecuredNASPDU(ue *context.UEContext, msgType uint8, payload []byte, securityHeaderType uint8) ([]byte, error) {
	if securityHeaderType == SecurityHeaderTypePlainNAS {
		pdu := &NASPDU{
			ProtocolDiscriminator: ProtocolDiscriminator5GMM,
			SecurityHeaderType:    SecurityHeaderTypePlainNAS,
			MessageType:           msgType,
			Payload:               payload,
		}
		return EncodeNASPDU(pdu), nil
	}

	plainPDU := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           msgType,
		Payload:               payload,
	}
	plainData := EncodeNASPDU(plainPDU)

	ue.DLCount++
	seqNum := uint8(ue.DLCount & 0xff)

	secPDU := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    securityHeaderType,
		SequenceNumber:        seqNum,
		MessageType:           msgType,
		Payload:               payload,
	}

	if securityHeaderType == SecurityHeaderTypeIntegrityProtectedAndCiphered ||
		securityHeaderType == SecurityHeaderTypeIntegrityProtectedAndCipheredWithNewContext {
		ciphered, err := Cipher(ue, plainData, ue.DLCount)
		if err != nil {
			return nil, err
		}
		secPDU.Payload = ciphered[1:]
	}

	mac, err := CalculateMAC(ue, secPDU, ue.DLCount, 1)
	if err != nil {
		return nil, err
	}
	secPDU.MAC = mac

	return EncodeNASPDU(secPDU), nil
}

func DecodeSecuredNASPDU(ue *context.UEContext, data []byte) (*NASPDU, error) {
	pdu, err := DecodeNASPDU(data)
	if err != nil {
		return nil, err
	}

	if pdu.SecurityHeaderType == SecurityHeaderTypePlainNAS {
		return pdu, nil
	}

	ue.ULCount++
	count := ue.ULCount

	mac, err := CalculateMAC(ue, pdu, count, 0)
	if err != nil {
		return nil, err
	}

	if !hmac.Equal(mac, pdu.MAC) {
		return nil, fmt.Errorf("MAC verification failed")
	}

	if pdu.SecurityHeaderType == SecurityHeaderTypeIntegrityProtectedAndCiphered ||
		pdu.SecurityHeaderType == SecurityHeaderTypeIntegrityProtectedAndCipheredWithNewContext {

		firstByte := (pdu.ProtocolDiscriminator << 4) | SecurityHeaderTypePlainNAS
		cipheredMsg := append([]byte{firstByte}, pdu.Payload...)

		decrypted, err := Decipher(ue, cipheredMsg, count)
		if err != nil {
			return nil, err
		}

		return DecodeNASPDU(decrypted)
	}

	return pdu, nil
}

func CalculateMAC(ue *context.UEContext, pdu *NASPDU, count uint32, direction uint8) ([]byte, error) {
	if ue.SecurityContext.IntegrityAlgorithm == AlgorithmNIA0 {
		return make([]byte, 4), nil
	}

	msg := make([]byte, 0)
	msg = append(msg, uint8(count>>24))
	msg = append(msg, uint8(count>>16))
	msg = append(msg, uint8(count>>8))
	msg = append(msg, uint8(count))
	msg = append(msg, direction)

	firstByte := (pdu.ProtocolDiscriminator << 4) | (pdu.SecurityHeaderType & 0x0f)
	msg = append(msg, firstByte)

	if pdu.SecurityHeaderType != SecurityHeaderTypePlainNAS {
		msg = append(msg, pdu.SequenceNumber)
	}

	msg = append(msg, pdu.MessageType)
	msg = append(msg, pdu.Payload...)

	switch ue.SecurityContext.IntegrityAlgorithm {
	case AlgorithmNIA1, AlgorithmNIA2, AlgorithmNIA3:
		return NIA2(ue.SecurityContext.KnasInt, count, direction, msg), nil
	default:
		return nil, fmt.Errorf("unsupported integrity algorithm: %d", ue.SecurityContext.IntegrityAlgorithm)
	}
}

func NIA2(key []byte, count uint32, direction uint8, msg []byte) []byte {
	if len(key) < 16 {
		key = append(key, make([]byte, 16-len(key))...)
	}

	m := make([]byte, 8)
	binary.BigEndian.PutUint32(m[0:4], count)
	m[4] = direction << 3
	binary.BigEndian.PutUint32(m[4:8], uint32(len(msg)*8))

	msgPadded := append(m, msg...)
	paddingLen := (16 - (len(msgPadded) % 16)) % 16
	if paddingLen > 0 {
		msgPadded = append(msgPadded, make([]byte, paddingLen)...)
	}

	block, _ := aes.NewCipher(key[:16])
	mac := make([]byte, 16)

	prev := make([]byte, 16)
	for i := 0; i < len(msgPadded); i += 16 {
		chunk := msgPadded[i : i+16]
		for j := 0; j < 16; j++ {
			prev[j] ^= chunk[j]
		}
		block.Encrypt(mac, prev)
		copy(prev, mac)
	}

	return mac[:4]
}

func Cipher(ue *context.UEContext, plaintext []byte, count uint32) ([]byte, error) {
	if ue.SecurityContext.CipheringAlgorithm == AlgorithmNEA0 {
		return plaintext, nil
	}

	switch ue.SecurityContext.CipheringAlgorithm {
	case AlgorithmNEA1, AlgorithmNEA2, AlgorithmNEA3:
		return NEA2(ue.SecurityContext.KnasEnc, count, 1, plaintext), nil
	default:
		return nil, fmt.Errorf("unsupported ciphering algorithm: %d", ue.SecurityContext.CipheringAlgorithm)
	}
}

func Decipher(ue *context.UEContext, ciphertext []byte, count uint32) ([]byte, error) {
	if ue.SecurityContext.CipheringAlgorithm == AlgorithmNEA0 {
		return ciphertext, nil
	}

	switch ue.SecurityContext.CipheringAlgorithm {
	case AlgorithmNEA1, AlgorithmNEA2, AlgorithmNEA3:
		return NEA2(ue.SecurityContext.KnasEnc, count, 0, ciphertext), nil
	default:
		return nil, fmt.Errorf("unsupported ciphering algorithm: %d", ue.SecurityContext.CipheringAlgorithm)
	}
}

func NEA2(key []byte, count uint32, bearer uint8, data []byte) []byte {
	if len(key) < 16 {
		key = append(key, make([]byte, 16-len(key))...)
	}

	block, _ := aes.NewCipher(key[:16])

	iv := make([]byte, 16)
	binary.BigEndian.PutUint32(iv[0:4], count)
	iv[4] = bearer << 3
	binary.BigEndian.PutUint32(iv[8:12], count)
	iv[12] = bearer << 3

	stream := cipher.NewCTR(block, iv)
	output := make([]byte, len(data))
	stream.XORKeyStream(output, data)

	return output
}

func DeriveNASKeys(ue *context.UEContext, kausf []byte) error {
	if len(kausf) < 32 {
		return fmt.Errorf("invalid KAUSF length")
	}

	kamf := KDF(kausf, []byte{0x6a}, []byte("KAMF"))
	if len(kamf) < 32 {
		return fmt.Errorf("failed to derive KAMF")
	}

	ue.SecurityContext.Kamf = kamf[:32]

	knasEnc := KDF(kamf, []byte{0x69}, []byte("NAS-enc"))
	if len(knasEnc) >= 16 {
		ue.SecurityContext.KnasEnc = knasEnc[:16]
	}

	knasInt := KDF(kamf, []byte{0x6a}, []byte("NAS-int"))
	if len(knasInt) >= 16 {
		ue.SecurityContext.KnasInt = knasInt[:16]
	}

	return nil
}

func KDF(key []byte, fc []byte, p0 []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(fc)

	p0LenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(p0LenBytes, uint16(len(p0)))
	mac.Write(p0LenBytes)
	mac.Write(p0)

	return mac.Sum(nil)
}

func ParseUESecurityCapabilities(data []byte) *context.UeSecurityCapability {
	if len(data) < 2 {
		return &context.UeSecurityCapability{
			NrEncryptionAlgs: []int{},
			NrIntegrityAlgs:  []int{},
		}
	}

	capability := &context.UeSecurityCapability{
		NrEncryptionAlgs:    []int{},
		NrIntegrityAlgs:     []int{},
		EutraEncryptionAlgs: []int{},
		EutraIntegrityAlgs:  []int{},
	}

	eaByte := data[0]
	for i := 7; i >= 0; i-- {
		if (eaByte & (1 << i)) != 0 {
			capability.NrEncryptionAlgs = append(capability.NrEncryptionAlgs, 7-i)
		}
	}

	iaByte := data[1]
	for i := 7; i >= 0; i-- {
		if (iaByte & (1 << i)) != 0 {
			capability.NrIntegrityAlgs = append(capability.NrIntegrityAlgs, 7-i)
		}
	}

	if len(data) >= 4 {
		eutraEaByte := data[2]
		for i := 7; i >= 0; i-- {
			if (eutraEaByte & (1 << i)) != 0 {
				capability.EutraEncryptionAlgs = append(capability.EutraEncryptionAlgs, 7-i)
			}
		}

		eutraIaByte := data[3]
		for i := 7; i >= 0; i-- {
			if (eutraIaByte & (1 << i)) != 0 {
				capability.EutraIntegrityAlgs = append(capability.EutraIntegrityAlgs, 7-i)
			}
		}
	}

	return capability
}

func SelectSecurityAlgorithms(ueCapability *context.UeSecurityCapability, amfSupportedEA []int, amfSupportedIA []int) (cipheringAlg int, integrityAlg int) {
	cipheringAlg = AlgorithmNEA0
	integrityAlg = AlgorithmNIA0

	preferenceOrderEA := []int{AlgorithmNEA2, AlgorithmNEA1, AlgorithmNEA3, AlgorithmNEA0}
	preferenceOrderIA := []int{AlgorithmNIA2, AlgorithmNIA1, AlgorithmNIA3, AlgorithmNIA0}

	for _, preferredEA := range preferenceOrderEA {
		if contains(amfSupportedEA, preferredEA) && contains(ueCapability.NrEncryptionAlgs, preferredEA) {
			cipheringAlg = preferredEA
			break
		}
	}

	for _, preferredIA := range preferenceOrderIA {
		if contains(amfSupportedIA, preferredIA) && contains(ueCapability.NrIntegrityAlgs, preferredIA) {
			integrityAlg = preferredIA
			break
		}
	}

	return cipheringAlg, integrityAlg
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
