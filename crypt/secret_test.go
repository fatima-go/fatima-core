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
	"github.com/fatima-go/fatima-core"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestResolveSecret(t *testing.T) {

}

func TestSecretDecryptB64(t *testing.T) {
	sample := "hello fatima-go"
	b64encoded := encryptBase64(sample)
	assert.EqualValues(t, "aGVsbG8gZmF0aW1hLWdv", b64encoded)
	assert.EqualValues(t, "hello fatima-go", secretDecryptB64(b64encoded))
}

func TestSecretDecryptNative(t *testing.T) {
	//
	//var cipherKeyByteFromProfile []byte
	//var cipherIVKeyByteFromProfile []byte
	profile := []byte(os.Getenv(fatima.ENV_FATIMA_PROFILE))
	fmt.Printf("%v\n", os.Getenv(fatima.ENV_FATIMA_PROFILE))
	fmt.Printf("%v\n", profile)
	fmt.Printf("%v\n", cipherKeyByteFromProfile)
	fmt.Printf("%v\n", cipherIVKeyByteFromProfile)
}

func encryptBase64(src string) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}
