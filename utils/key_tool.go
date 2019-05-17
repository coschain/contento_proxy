package utils

import (
	"github.com/coschain/contentos-go/prototype"
	"hash/crc32"
	"math/rand"
	"time"
)

const (
	leftPos = 0
	rightPos = 8
	randomLen = 8
 	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateNewKey() (string, string, error) {
	privKey, err := prototype.GenerateNewKey()
	if err != nil {
		return "", "", err
	}
	pubKey, err := privKey.PubKey()
	if err != nil {
		return "", "", err
	}
	privKeyStr := privKey.ToWIF()
	pubKeyStr := pubKey.ToWIF()
	return pubKeyStr, privKeyStr, nil
}

func GenerateUUID(content string) uint64 {
	crc32q := crc32.MakeTable(0xD5828281)
	randContent := content + string(rand.Intn(1e5))
	return uint64(time.Now().Unix()*1e9) + uint64(crc32.Checksum([]byte(randContent), crc32q))
}

func GenerateName(name string) string {
	newName := name
	if len(newName) > rightPos {
		newName = newName[leftPos:rightPos]
	}
	randomStr := RandStringBytes(randomLen)

	newName += randomStr
	return newName
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}