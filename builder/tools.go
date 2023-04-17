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
 * @project fatima-core
 * @author dave
 * @date 23. 4. 12. 오후 1:41
 */

package builder

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

/**
 * @author jin.freestyle@gmail.com
 *
 */

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

// readProperties read properties (key=value pairs)
func readProperties(path string) (map[string]string, error) {
	resolved := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var line string
	var idx int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line = strings.Trim(scanner.Text(), " ")
		if strings.HasPrefix(line, "#") || len(line) < 3 {
			continue
		}
		idx = strings.Index(line, "#")
		if idx > 0 {
			if line[idx-1] == ' ' {
				line = line[:idx]
			}
		}
		idx = strings.Index(line, "=")
		if idx < 1 {
			continue
		}
		resolved[line[:idx]] = line[idx+1:]
	}

	return resolved, nil
}
