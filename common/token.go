package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Payload struct {
	UserID   string `json:"userID"`
	IsAdmin  bool   `json:"isAdmin"`
	Username string `json:"username"`
}

func GenerateToken(id string, isAdmin bool, username string) (string, error) {
	secretKey := os.Getenv("TOKEN_SECRET_KEY")

	payload := Payload{
		UserID:   id,
		IsAdmin:  isAdmin,
		Username: username,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// 添加时间戳使得每次都不一样
	timestamp := time.Now().Unix()
	data = append(data, []byte(fmt.Sprintf(":%d", timestamp))...)

	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func ParseToken(token string) (Payload, error) {
	secretKey := os.Getenv("TOKEN_SECRET_KEY")
	ciphertext, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return Payload{}, err
	}

	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return Payload{}, err
	}

	if len(ciphertext) < aes.BlockSize {
		return Payload{}, fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	stream.XORKeyStream(ciphertext, ciphertext)

	// 移除时间戳
	parts := []byte(string(ciphertext))
	data := parts[:len(parts)-11]

	var payload Payload
	err = json.Unmarshal(data, &payload)
	if err != nil {
		return Payload{}, err
	}

	return payload, nil
}
func ExtractToken(bearerString string) string {
	prefix := "Bearer "
	if strings.HasPrefix(bearerString, prefix) {
		return bearerString[len(prefix):]
	}
	return ""
}
