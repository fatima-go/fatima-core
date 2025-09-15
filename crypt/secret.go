/*
 * Copyright 2023 github.com/fatima-go
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @project fatima-go
 * @author dave_01
 * @date 23. 7. 18. 오후 4:31
 */

package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/fatima-go/fatima-core"
	log "github.com/fatima-go/fatima-log"
)

const (
	SecretSchemeNative = "native"
	SecretSchemeB64    = "b64"
	SecretSchemeAWS    = "aws"
)

// CreateSecretBase64 b64 스킴으로 암호화
func CreateSecretBase64(src string) string {
	return fmt.Sprintf("%s:%s", SecretSchemeB64, secretEncryptB64(src))
}

// CreateSecretNative native 스킴으로 암호화
func CreateSecretNative(src string) string {
	return fmt.Sprintf("%s:%s", SecretSchemeNative, secretEncryptNative(src))
}

// ResolveSecret resolve secret(with scheme) string
func ResolveSecret(src string) string {
	if len(src) == 0 {
		return src
	}

	idx := strings.Index(src, ":")
	if idx < 1 || (len(src) <= idx+1) {
		// empty scheme or content
		return src
	}

	scheme := src[:idx]
	secret := src[idx+1:]

	switch scheme {
	case SecretSchemeNative:
		return secretDecryptFuncNative(secret)
	case SecretSchemeB64:
		return secretDecryptFuncB64(secret)
	case SecretSchemeAWS:
		return secretDecryptFuncAWS(secret)
	}

	log.Warn("scheme %s is not supported", scheme)
	return src
}

type SecretDecryptFunc func(string) string

var secretDecryptFuncNative = secretDecryptNative
var secretDecryptFuncB64 = secretDecryptB64
var secretDecryptFuncAWS = secretDecryptAWS

func SetSecretDecryptFunc(schemeName string, decryptFunc SecretDecryptFunc) error {
	switch schemeName {
	case SecretSchemeNative:
		secretDecryptFuncNative = decryptFunc
	case SecretSchemeB64:
		secretDecryptFuncB64 = decryptFunc
	case SecretSchemeAWS:
		secretDecryptFuncAWS = decryptFunc
	default:
		return fmt.Errorf("invalid secret scheme %s", schemeName)
	}
	return nil
}

var cipherKeyByteFromProfile []byte
var cipherIVKeyByteFromProfile []byte

const (
	CipherKeyBytesLength = 32
	CipherIVKeyLen       = 16
)

func init() {
	// example : dev, qa, prod
	envProfile := os.Getenv(fatima.ENV_FATIMA_PROFILE)
	if len(envProfile) == 0 {
		envProfile = "LOCAL"
	}
	profile := []byte(strings.ToLower(envProfile))
	cipherKeyByteFromProfile = make([]byte, CipherKeyBytesLength)
	validLen := len(profile)
	if validLen > CipherKeyBytesLength {
		validLen = CipherKeyBytesLength
	}
	copy(cipherKeyByteFromProfile, profile[:validLen])
	cipherIVKeyByteFromProfile = cipherKeyByteFromProfile[:CipherIVKeyLen]
}

func secretDecryptNative(src string) string {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		log.Warn("cipher [%s] is invalid base64 format : %s", src, err.Error())
		return src
	}

	cipherBlock, err := aes.NewCipher(cipherKeyByteFromProfile)
	if err != nil {
		log.Warn("creating cipher error. key=%v, error=%s", cipherKeyByteFromProfile, err.Error())
		return src
	}

	//goland:noinspection SpellCheckingInspection
	cbcDecryptor := cipher.NewCBCDecrypter(cipherBlock, cipherIVKeyByteFromProfile)
	plaintextBytes := make([]byte, len(ciphertextBytes))

	cbcDecryptor.CryptBlocks(plaintextBytes, ciphertextBytes)
	return string(unpad(plaintextBytes))
}

func secretEncryptNative(src string) string {
	cipherBlock, err := aes.NewCipher(cipherKeyByteFromProfile)
	if err != nil {
		log.Warn("NewCipher error [%s] : %s", cipherKeyByteFromProfile, err.Error())
		return src
	}

	cbcEncryptor := cipher.NewCBCEncrypter(cipherBlock, cipherIVKeyByteFromProfile)
	paddedPlaintextBytes := pad([]byte(src), cbcEncryptor.BlockSize())

	ciphertextBytes := make([]byte, len(paddedPlaintextBytes))
	cbcEncryptor.CryptBlocks(ciphertextBytes, paddedPlaintextBytes)
	return base64.StdEncoding.EncodeToString(ciphertextBytes)
}

func pad(blocks []byte, blockSize int) []byte {
	padLen := blockSize - len(blocks)%blockSize
	padBlocks := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(blocks, padBlocks...)
}

func unpad(blocks []byte) []byte {
	blockLen := len(blocks)
	paddedLen := int(blocks[blockLen-1])
	return blocks[:(blockLen - paddedLen)]
}

func secretDecryptB64(src string) string {
	decoded, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		log.Warn("cipher [%s] base64 decoding error : %s", src, err.Error())
		return src
	}
	return string(decoded)
}

func secretEncryptB64(src string) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}

func secretDecryptAWS(src string) string {
	log.Warn("not support aws secret manager")
	return src
}
