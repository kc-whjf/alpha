package pbe

import (
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"github.com/kc-whjf/alpha/autil/acrypto"
)

func getDerivedKey(password string, salt []byte, count, l int) ([]byte, error) {
	derived := md5.Sum([]byte(password + string(salt)))
	for i := 0; i < count-1; i++ {
		derived = md5.Sum(derived[:])
	}
	return derived[:l], nil
}

func PBEWithMD5AndDES_Decrypt(msg, password string) (string, error) {
	msgBytes, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return "", err
	}
	salt := msgBytes[:des.BlockSize]
	encText := msgBytes[des.BlockSize:]

	derived, err := getDerivedKey(password, salt, 1000, des.BlockSize*2)
	if err != nil {
		return "", err
	}

	key := derived[:des.BlockSize]
	iv := derived[des.BlockSize:]

	text, err := acrypto.DesCBCDecrypt(encText, key, iv)
	if err != nil {
		return "", err
	}
	return string(text), nil
}

func PBEWithMD5AndDES_Encrypt(msg, password string) (string, error) {
	salt := make([]byte, des.BlockSize)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	derived, err := getDerivedKey(password, salt, 1000, des.BlockSize*2)
	if err != nil {
		return "", err
	}
	key := derived[:des.BlockSize]
	iv := derived[des.BlockSize:]

	encText, err := acrypto.DesCBCEncrypt([]byte(msg), key, iv)
	if err != nil {
		return "", err
	}
	r := append(salt, encText...)
	encodeString := base64.StdEncoding.EncodeToString(r)
	return encodeString, nil
}
