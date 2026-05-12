package sealer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type Sealer struct {
	aead cipher.AEAD
}

func New(key []byte) (*Sealer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Sealer{aead: gcm}, nil
}

func (s *Sealer) Encrypt(text []byte) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := s.aead.Seal(nil, nonce, text, nil)
	return append(nonce, ciphertext...), nil
}

func (s *Sealer) Decrypt(data []byte) ([]byte, error) {
	nonceSize := s.aead.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	text, err := s.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return text, nil
}
