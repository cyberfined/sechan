package proto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"
)

const (
	ProtocolVersion  = 1
	MaxMessagesCount = 0xffffffff
)

var (
	ErrCounterOverflow = errors.New("message counter overflow")
	ErrLowCounter      = errors.New("message counter is too low")
	ErrShortMessage    = errors.New("message is too short")
	ErrAuth            = errors.New("authentication failed")
	ErrIVGen           = errors.New("failed to generate initialization vector")
)

type CryptoState struct {
	SendEnc    cipher.Block
	RecEnc     cipher.Block
	SendAuth   hash.Hash
	RecAuth    hash.Hash
	MsgSendCtr uint32
	MsgRecCtr  uint32
}

func InitCryptoState(key []byte, isUserA bool) *CryptoState {
	cs := &CryptoState{}

	KeySendEnc := sha256d(append(key, []byte("Enc from A to B")...))
	KeyRecEnc := sha256d(append(key, []byte("Enc from B to A")...))
	KeySendAuth := sha256d(append(key, []byte("Auth from A to B")...))
	KeyRecAuth := sha256d(append(key, []byte("Auth from B to A")...))

	if !isUserA {
		KeySendEnc, KeyRecEnc = KeyRecEnc, KeySendEnc
		KeySendAuth, KeyRecAuth = KeyRecAuth, KeySendAuth
	}

	cs.SendEnc, _ = aes.NewCipher(KeySendEnc)
	cs.RecEnc, _ = aes.NewCipher(KeyRecEnc)
	cs.SendAuth = hmac.New(sha256.New, KeySendAuth)
	cs.RecAuth = hmac.New(sha256.New, KeyRecAuth)
	cs.MsgSendCtr = 0
	cs.MsgRecCtr = 0

	return cs
}

func (cs *CryptoState) ExchangeKeys() {
	cs.SendEnc, cs.RecEnc = cs.RecEnc, cs.SendEnc
	cs.SendAuth, cs.RecAuth = cs.RecAuth, cs.SendAuth
}

func (cs *CryptoState) AuthAndEncrypt(data []byte) ([]byte, error) {
	if cs.MsgSendCtr == MaxMessagesCount {
		return nil, ErrCounterOverflow
	}
	cs.MsgSendCtr++

	iv := make([]byte, cs.SendEnc.BlockSize())
	binary.LittleEndian.PutUint32(iv, cs.MsgSendCtr)
	_, err := rand.Read(iv[4:])
	if err != nil {
		return nil, ErrIVGen
	}
	ctr := cipher.NewCTR(cs.SendEnc, iv)

	auth := cs.SendAuthMessage(data)
	cipherText := make([]byte, len(iv)+len(auth))
	copy(cipherText, iv)
	ctr.XORKeyStream(cipherText[len(iv):], auth)
	return cipherText, nil
}

func (cs *CryptoState) DecryptAndAuth(data []byte) ([]byte, error) {
	if cs.MsgRecCtr == MaxMessagesCount {
		return nil, ErrCounterOverflow
	}

	blockSize := cs.RecEnc.BlockSize()
	hashSize := cs.RecAuth.Size()
	if len(data) <= blockSize+hashSize {
		return nil, ErrShortMessage
	}

	iv := data[:blockSize]
	data = data[blockSize:]
	counter := binary.LittleEndian.Uint32(iv[:4])
	ctr := cipher.NewCTR(cs.RecEnc, iv)

	decr := make([]byte, len(data))
	ctr.XORKeyStream(decr, data)

	plainText := decr[:len(decr)-hashSize]
	hash := cs.RecAuthMessage(plainText)
	if !hmac.Equal(hash, decr) {
		return nil, ErrAuth
	}

	if counter <= cs.MsgRecCtr {
		return nil, ErrLowCounter
	}
	cs.MsgRecCtr = counter

	return plainText, nil
}

func (cs *CryptoState) SendAuthMessage(data []byte) []byte {
	metadata := getMetadata()
	cs.SendAuth.Write(metadata)
	cs.SendAuth.Write(data)
	return cs.SendAuth.Sum(data)
}

func (cs *CryptoState) RecAuthMessage(data []byte) []byte {
	metadata := getMetadata()
	cs.RecAuth.Write(metadata)
	cs.RecAuth.Write(data)
	return cs.RecAuth.Sum(data)
}

func getMetadata() []byte {
	metadata := make([]byte, 4)
	binary.LittleEndian.PutUint32(metadata, ProtocolVersion)
	return metadata
}

func sha256d(data []byte) []byte {
	result := sha256.Sum256(data)
	result = sha256.Sum256(result[:])
	return result[:]
}
