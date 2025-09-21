package utils

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/rand"
	"strings"
)

type Address = string

const charset = "0123456789abcdef"

func RandomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

func RandomBytes(n int) []byte {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return []byte(sb.String())
}
func RandomAddress() common.Address {
	//fmt.Println(RandomString(40))

	return common.HexToAddress(RandomString(40))
}
func RandomHash() common.Hash {
	//fmt.Println(RandomString(64))
	return common.HexToHash(RandomString(64))
}

func UInt64ToBytes(num uint64) []byte {
	buf := make([]byte, 8)
	for i := uint(0); i < 8; i++ {
		buf[i] = byte(num >> (i * 8))
	}
	return buf
}

func BytesToUInt64(buf []byte) uint64 {
	var num uint64
	for i := uint(0); i < 8; i++ {
		num |= uint64(buf[i]) << (i * 8)
	}
	return num
}
func GenerateEthereumAddress() common.Address {
	// 生成新的私钥（ECDSA）
	privateKey, _ := crypto.GenerateKey()
	//if err != nil {
	//	return "", "", "", err
	//}

	// 获取私钥的字节表示
	//privateKeyBytes := crypto.FromECDSA(privateKey)
	//privateKeyHex = fmt.Sprintf("%x", privateKeyBytes)

	// 从私钥派生公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)

	// 获取公钥的字节表示
	//publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	//publicKeyHex = fmt.Sprintf("%x", publicKeyBytes)
	//
	// 生成以太坊地址
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	//addressHex = address.Hex()

	return address
}
