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

const (
	BuiltinVariableHome           = "${var.builtin.user.home}"
	BuiltinVariableFatimaHome     = "${var.builtin.fatima.home}"
	BuiltinVariableLocalIpaddress = "${var.builtin.local.ipaddress}"
	BuiltinVariableYyyymm         = "${var.builtin.date.yyyymm}"
	BuiltinVariableYyyymmdd       = "${var.builtin.date.yyyymmdd}"
	BuiltinVariableAppName        = "${var.builtin.app.name}"
	BuiltinVariableAppFolderData  = "${var.builtin.app.folder.data}"

	GlobalDefinePackageHostname  = "var.global.package.hostname"
	GlobalDefinePackageGroupname = "var.global.package.groupname"
	GlobalDefinePackageName      = "var.global.package.name"
)

const (
	GofatimaPropPprofAddress = "gofatima.pprof.address"    // e.g :6060, localhost:6060
	GofatimaRedirectConsole  = "gofatima.redirect.console" // e.g true, false. default=true
)
