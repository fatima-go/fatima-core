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
 * @date 23. 7. 18. 오후 5:24
 */

package crypt

import (
	"encoding/base64"
	"fmt"
	"testing"

	log "github.com/fatima-go/fatima-log"
	"github.com/stretchr/testify/assert"
)

var sample = "hello fatima-go"

func TestSecretDecryptB64(t *testing.T) {
	b64encoded := encryptBase64(sample)
	assert.EqualValues(t, "aGVsbG8gZmF0aW1hLWdv", b64encoded)
	assert.EqualValues(t, "hello fatima-go", secretDecryptB64(b64encoded))
}

func TestSecretDecryptNative(t *testing.T) {
	log.Initialize(log.NewPreference(""))
	cipherText := secretEncryptNative(sample)
	decoded := secretDecryptNative(cipherText)
	assert.EqualValues(t, sample, decoded)
}

func TestResolveSecret(t *testing.T) {
	log.Initialize(log.NewPreference(""))
	cipherText := secretEncryptNative(sample)
	decoded := ResolveSecret(fmt.Sprintf("%s:%s", SecretSchemeNative, cipherText))
	assert.EqualValues(t, sample, decoded)
	cipherText = secretEncryptB64(sample)
	decoded = ResolveSecret(fmt.Sprintf("%s:%s", SecretSchemeB64, cipherText))
	assert.EqualValues(t, sample, decoded)
	decoded = ResolveSecret(fmt.Sprintf("%s:%s", SecretSchemeAWS, sample))
	assert.EqualValues(t, sample, decoded)
}

func TestCreateAndResolveSecret(t *testing.T) {
	log.Initialize(log.NewPreference(""))
	cipherText := CreateSecretBase64(sample)
	assert.EqualValues(t, sample, ResolveSecret(cipherText))
	cipherText = CreateSecretNative(sample)
	assert.EqualValues(t, sample, ResolveSecret(cipherText))
}

func TestSetSecretDecryptFunc(t *testing.T) {
	log.Initialize(log.NewPreference(""))
	_ = SetSecretDecryptFunc(SecretSchemeNative, exampleSecretDecryptNative)
	_ = SetSecretDecryptFunc(SecretSchemeB64, exampleSecretDecryptB64)
	_ = SetSecretDecryptFunc(SecretSchemeAWS, exampleSecretDecryptAWS)
	assert.EqualValues(t, "NATIVE", ResolveSecret(fmt.Sprintf("%s:%s", SecretSchemeNative, sample)))
	assert.EqualValues(t, "B64", ResolveSecret(fmt.Sprintf("%s:%s", SecretSchemeB64, sample)))
	assert.EqualValues(t, "AWS", ResolveSecret(fmt.Sprintf("%s:%s", SecretSchemeAWS, sample)))
	assert.NotNil(t, SetSecretDecryptFunc("unknownScheme", exampleSecretDecryptAWS))
}

func exampleSecretDecryptAWS(src string) string {
	return "AWS"
}

func exampleSecretDecryptB64(src string) string {
	return "B64"
}

func exampleSecretDecryptNative(src string) string {
	return "NATIVE"
}

func encryptBase64(src string) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}
