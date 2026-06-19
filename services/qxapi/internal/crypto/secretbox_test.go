package crypto

import (
	"encoding/base64"
	"errors"
	"testing"
)

func testMasterKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestEncryptDecrypt(t *testing.T) {
	enc, err := NewEncryptor(testMasterKey(t))
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	cipher, err := enc.Encrypt([]byte("secret-key"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plain, err := enc.Decrypt(cipher)
	if err != nil || string(plain) != "secret-key" {
		t.Fatalf("decrypt: %v %q", err, plain)
	}
}

func TestInvalidKey(t *testing.T) {
	if _, err := NewEncryptor(""); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("empty key: %v", err)
	}
	if _, err := NewEncryptor("not-base64!!!"); err == nil {
		t.Fatal("expected decode error")
	}
	short := base64.StdEncoding.EncodeToString([]byte("short"))
	if _, err := NewEncryptor(short); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("short key: %v", err)
	}
}
