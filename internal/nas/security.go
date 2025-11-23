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
	case AlgorithmNIA1:
		return NIA1(ue.SecurityContext.KnasInt, count, direction, msg), nil
	case AlgorithmNIA2:
		return NIA2(ue.SecurityContext.KnasInt, count, direction, msg), nil
	case AlgorithmNIA3:
		return NIA3(ue.SecurityContext.KnasInt, count, direction, msg), nil
	default:
		return nil, fmt.Errorf("unsupported integrity algorithm: %d", ue.SecurityContext.IntegrityAlgorithm)
	}
}

func NIA1(key []byte, count uint32, direction uint8, msg []byte) []byte {
	if len(key) < 16 {
		key = append(key, make([]byte, 16-len(key))...)
	}

	iv := make([]byte, 16)
	binary.BigEndian.PutUint32(iv[0:4], count)
	iv[4] = direction << 3
	iv[8] = direction << 3
	binary.BigEndian.PutUint32(iv[12:16], count)

	keystream := snow3gKeystream(key[:16], iv, len(msg)+8)

	mac := uint32(0)
	for i := 0; i < len(msg); i++ {
		mac ^= uint32(msg[i]) << (24 - (i%4)*8)
		if i%4 == 3 || i == len(msg)-1 {
			idx := (i / 4) * 4
			if idx+4 <= len(keystream) {
				ks := binary.BigEndian.Uint32(keystream[idx : idx+4])
				mac ^= ks
			}
			if i%4 == 3 {
				mac = 0
			}
		}
	}

	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, mac)
	return result
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

func NIA3(key []byte, count uint32, direction uint8, msg []byte) []byte {
	if len(key) < 16 {
		key = append(key, make([]byte, 16-len(key))...)
	}

	iv := make([]byte, 16)
	binary.BigEndian.PutUint32(iv[0:4], count)
	iv[4] = (direction << 3) & 0xf8
	iv[8] = iv[0]
	iv[9] = iv[1]
	iv[10] = iv[2]
	iv[11] = iv[3]
	iv[12] = iv[4]
	iv[13] = 0
	iv[14] = 0
	iv[15] = 0

	keystream := zucKeystream(key[:16], iv, len(msg)+4)

	mac := uint32(0)
	for i := 0; i < len(msg); i++ {
		if i%4 == 0 && i+4 <= len(keystream) {
			z := binary.BigEndian.Uint32(keystream[i : i+4])
			msgWord := uint32(0)
			for j := 0; j < 4 && i+j < len(msg); j++ {
				msgWord |= uint32(msg[i+j]) << (24 - j*8)
			}
			mac ^= (msgWord ^ z)
		}
	}

	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, mac)
	return result
}

func Cipher(ue *context.UEContext, plaintext []byte, count uint32) ([]byte, error) {
	if ue.SecurityContext.CipheringAlgorithm == AlgorithmNEA0 {
		return plaintext, nil
	}

	switch ue.SecurityContext.CipheringAlgorithm {
	case AlgorithmNEA1:
		return NEA1(ue.SecurityContext.KnasEnc, count, 1, plaintext), nil
	case AlgorithmNEA2:
		return NEA2(ue.SecurityContext.KnasEnc, count, 1, plaintext), nil
	case AlgorithmNEA3:
		return NEA3(ue.SecurityContext.KnasEnc, count, 1, plaintext), nil
	default:
		return nil, fmt.Errorf("unsupported ciphering algorithm: %d", ue.SecurityContext.CipheringAlgorithm)
	}
}

func Decipher(ue *context.UEContext, ciphertext []byte, count uint32) ([]byte, error) {
	if ue.SecurityContext.CipheringAlgorithm == AlgorithmNEA0 {
		return ciphertext, nil
	}

	switch ue.SecurityContext.CipheringAlgorithm {
	case AlgorithmNEA1:
		return NEA1(ue.SecurityContext.KnasEnc, count, 0, ciphertext), nil
	case AlgorithmNEA2:
		return NEA2(ue.SecurityContext.KnasEnc, count, 0, ciphertext), nil
	case AlgorithmNEA3:
		return NEA3(ue.SecurityContext.KnasEnc, count, 0, ciphertext), nil
	default:
		return nil, fmt.Errorf("unsupported ciphering algorithm: %d", ue.SecurityContext.CipheringAlgorithm)
	}
}

func NEA1(key []byte, count uint32, bearer uint8, data []byte) []byte {
	if len(key) < 16 {
		key = append(key, make([]byte, 16-len(key))...)
	}

	iv := make([]byte, 16)
	binary.BigEndian.PutUint32(iv[0:4], count)
	iv[4] = (bearer << 3) | uint8((count>>29)&0x07)
	iv[8] = iv[4]
	binary.BigEndian.PutUint32(iv[12:16], count)

	keystream := snow3gKeystream(key[:16], iv, len(data))

	output := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		output[i] = data[i] ^ keystream[i]
	}

	return output
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

func NEA3(key []byte, count uint32, bearer uint8, data []byte) []byte {
	if len(key) < 16 {
		key = append(key, make([]byte, 16-len(key))...)
	}

	iv := make([]byte, 16)
	binary.BigEndian.PutUint32(iv[0:4], count)
	iv[4] = (bearer << 3) & 0xf8
	iv[8] = iv[0]
	iv[9] = iv[1]
	iv[10] = iv[2]
	iv[11] = iv[3]
	iv[12] = iv[4]
	iv[13] = 0
	iv[14] = 0
	iv[15] = 0

	keystream := zucKeystream(key[:16], iv, len(data))

	output := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		output[i] = data[i] ^ keystream[i]
	}

	return output
}

func snow3gKeystream(key []byte, iv []byte, length int) []byte {
	s := &snow3gState{}
	s.initialize(key, iv)

	keystream := make([]byte, length)
	for i := 0; i < length; i += 4 {
		z := s.generateKeyword()
		binary.BigEndian.PutUint32(keystream[i:], z)
	}

	return keystream
}

type snow3gState struct {
	lfsr [16]uint32
	r1   uint32
	r2   uint32
	r3   uint32
}

func (s *snow3gState) initialize(key []byte, iv []byte) {
	for i := 0; i < 16; i++ {
		s.lfsr[i] = uint32(key[i%16])<<24 | uint32(iv[i%16])<<16
	}

	s.r1 = 0
	s.r2 = 0
	s.r3 = 0

	for i := 0; i < 32; i++ {
		s.clockFSM(s.clockLFSR())
	}
}

func (s *snow3gState) clockLFSR() uint32 {
	v := s.lfsr[0]
	s0 := s.lfsr[0]
	s11 := s.lfsr[11]

	newS15 := s0 ^ s11 ^ (s.lfsr[2] >> 8) ^ (s.lfsr[13] << 8)

	for i := 0; i < 15; i++ {
		s.lfsr[i] = s.lfsr[i+1]
	}
	s.lfsr[15] = newS15

	return v
}

func (s *snow3gState) clockFSM(input uint32) uint32 {
	r := s.r1 + s.r2
	s.r3 = s.r2 + (s.lfsr[5] ^ input)
	s.r2 = s.r1
	s.r1 = r
	return r ^ s.r3
}

func (s *snow3gState) generateKeyword() uint32 {
	t := s.clockFSM(s.clockLFSR())
	return t ^ s.lfsr[0]
}

func zucKeystream(key []byte, iv []byte, length int) []byte {
	z := &zucState{}
	z.initialize(key, iv)

	keystream := make([]byte, length)
	for i := 0; i < length; i += 4 {
		w := z.generateKeyword()
		binary.BigEndian.PutUint32(keystream[i:], w)
	}

	return keystream
}

type zucState struct {
	lfsr [16]uint32
	r1   uint32
	r2   uint32
}

func (z *zucState) initialize(key []byte, iv []byte) {
	d := []uint32{
		0x44D7, 0x26BC, 0x626B, 0x135E, 0x5789, 0x35E2, 0x7135, 0x09AF,
		0x4D78, 0x2F13, 0x6BC4, 0x1AF1, 0x5E26, 0x3C4D, 0x789A, 0x47AC,
	}

	for i := 0; i < 16; i++ {
		z.lfsr[i] = (d[i]<<16 | uint32(key[i]))<<16 | uint32(iv[i])
	}

	z.r1 = 0
	z.r2 = 0

	for i := 0; i < 32; i++ {
		w := z.bitReorganization()
		z.nonlinearFunction(w)
		z.lfsrWithInitMode(w >> 1)
	}
}

func (z *zucState) bitReorganization() uint32 {
	x0 := ((z.lfsr[15] & 0x7FFF8000) << 1) | (z.lfsr[14] & 0xFFFF)
	x1 := ((z.lfsr[11] & 0xFFFF) << 16) | (z.lfsr[9] >> 15)
	x2 := ((z.lfsr[7] & 0xFFFF) << 16) | (z.lfsr[5] >> 15)
	x3 := ((z.lfsr[2] & 0xFFFF) << 16) | (z.lfsr[0] >> 15)

	return x0 ^ x1 ^ x2 ^ x3
}

func (z *zucState) nonlinearFunction(x uint32) uint32 {
	w := (x + z.r1) ^ z.r2

	w1 := z.r1 + x
	w2 := z.r2 ^ z.r1

	z.r1 = z.s(z.l1((w1<<16)|(w1>>16)))
	z.r2 = z.s(z.l2((w2<<16)|(w2>>16)))

	return w
}

func (z *zucState) lfsrWithInitMode(u uint32) {
	v := z.lfsr[0]
	s16 := (v << 1) ^ (z.lfsr[0] >> 31) ^ (z.lfsr[13] >> 31) ^ u

	for i := 0; i < 15; i++ {
		z.lfsr[i] = z.lfsr[i+1]
	}
	z.lfsr[15] = s16
}

func (z *zucState) lfsrWithWorkMode() {
	v := z.lfsr[0]
	s16 := (v << 1) ^ (z.lfsr[0] >> 31) ^ (z.lfsr[13] >> 31)

	for i := 0; i < 15; i++ {
		z.lfsr[i] = z.lfsr[i+1]
	}
	z.lfsr[15] = s16
}

func (z *zucState) generateKeyword() uint32 {
	z.lfsrWithWorkMode()
	return z.nonlinearFunction(z.bitReorganization()) ^ z.lfsr[0]
}

func (z *zucState) s(x uint32) uint32 {
	sbox := []byte{
		0x3e, 0x72, 0x5b, 0x47, 0xca, 0xe0, 0x00, 0x33, 0x04, 0xd1, 0x54, 0x98, 0x09, 0xb9, 0x6d, 0xcb,
		0x7b, 0x1b, 0xf9, 0x32, 0xaf, 0x9d, 0x6a, 0xa5, 0xb8, 0x2d, 0xfc, 0x1d, 0x08, 0x53, 0x03, 0x90,
	}

	b0 := byte(x >> 24)
	b1 := byte(x >> 16)
	b2 := byte(x >> 8)
	b3 := byte(x)

	if int(b0>>4) < len(sbox) && int(b1>>4) < len(sbox) && int(b2>>4) < len(sbox) && int(b3>>4) < len(sbox) {
		return uint32(sbox[b0>>4])<<24 | uint32(sbox[b1>>4])<<16 | uint32(sbox[b2>>4])<<8 | uint32(sbox[b3>>4])
	}

	return x
}

func (z *zucState) l1(x uint32) uint32 {
	return x ^ ((x << 2) | (x >> 30)) ^ ((x << 10) | (x >> 22)) ^ ((x << 18) | (x >> 14)) ^ ((x << 24) | (x >> 8))
}

func (z *zucState) l2(x uint32) uint32 {
	return x ^ ((x << 8) | (x >> 24)) ^ ((x << 14) | (x >> 18)) ^ ((x << 22) | (x >> 10)) ^ ((x << 30) | (x >> 2))
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
