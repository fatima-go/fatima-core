//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// @project fatima
// @author DeockJin Chung (jin.freestyle@gmail.com)
// @date 2017. 3. 6. PM 7:42
//

package infra

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

func expressBytes(bytes uint64) string {
	value := float32(bytes)

	var stringValue string
	switch {
	case bytes >= TERABYTE:
		return fmt.Sprintf("%.2fT", value/TERABYTE)
	case bytes >= GIGABYTE:
		return fmt.Sprintf("%.2fG", value/GIGABYTE)
	case bytes >= MEGABYTE:
		stringValue = fmt.Sprintf("%.1f", value/MEGABYTE)
		return fmt.Sprintf("%sM", strings.TrimSuffix(stringValue, ".0"))
	case bytes >= KILOBYTE:
		return fmt.Sprintf("%dK", (int)(value/KILOBYTE))
	case bytes >= BYTE:
		return fmt.Sprintf("%dB", value)
	}

	return "0"
}

func isDirectory(path string) bool {
	if stat, err := os.Stat(path); err != nil {
		return stat.IsDir()
	}
	return false
}

func ensureDirectory(path string, forceCreate bool) error {
	if stat, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if forceCreate {
				return os.MkdirAll(path, 0755)
			}
		} else if !stat.IsDir() {
			return errors.New(fmt.Sprintf("%s path exist as file", path))
		}
	}

	return nil
}
