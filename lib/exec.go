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

package lib

import (
	"bytes"
	"errors"
	"os/exec"
	"regexp"
)

func ExecuteCommand(command string) (string, error) {
	if len(command) == 0 {
		return "", errors.New("empty command")
	}

	var cmd *exec.Cmd
	s := regexp.MustCompile("\\s+").Split(command, -1)
	i := len(s)
	if i == 0 {
		return "", errors.New("empty command")
	} else if i == 1 {
		cmd = exec.Command(s[0])
	} else {
		cmd = exec.Command(s[0], s[1:]...)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func ExecuteShell(command string) (string, error) {
	if len(command) == 0 {
		return "", errors.New("empty command")
	}

	var cmd *exec.Cmd
	cmd = exec.Command("/bin/sh", "-c", command)
	//s := regexp.MustCompile("\\s+").Split(command, -1)
	//i := len(s)
	//if i == 0 {
	//	return "", errors.New("empty command")
	//} else if i == 1 {
	//	cmd = exec.Command(s[0])
	//} else {
	//	cmd = exec.Command(s[0], s[1:]...)
	//}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
