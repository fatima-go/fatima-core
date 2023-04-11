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

package fatima

type SystemProc interface {
	GetPid() int
	GetUid() int
	GetProgramName() string
	GetUsername() string
	GetHomeDir() string
	GetGid() string
}

type FolderGuide interface {
	GetFatimaHome() string

	// GetPackageProcFile return package process configuration file ($FATIMA_HOME/conf/fatima-package.yaml)
	GetPackageProcFile() string
	GetAppProcFolder() string
	GetAppFolder() string
	GetLogFolder() string
	GetConfFolder() string
	GetDataFolder() string
	CreateTmpFolder() string
	CreateTmpFilePath() string
}

type FatimaEnv interface {
	GetSystemProc() SystemProc
	GetFolderGuide() FolderGuide
	GetProfile() string
}
