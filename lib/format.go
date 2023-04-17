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

import "fmt"

func FormatBytes(v int) string {
	if (v >> 30) > 0 {
		f := float64(v)
		f = f / (1024 * 1024 * 1024)
		return fmt.Sprintf("%.2fG", f)
	}
	if (v >> 20) > 0 {
		return fmt.Sprintf("%dM", v>>20)
	}
	return fmt.Sprintf("%dK", v>>10)
}
