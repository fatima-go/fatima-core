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

package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/builder"
	"github.com/fatima-go/fatima-core/lib"
)

type MockFatimaEnv struct {
	systemProc  fatima.SystemProc
	folderGuide fatima.FolderGuide
	profile     string
}

func NewMockFatimaEnv(processName string) *MockFatimaEnv {
	env := new(MockFatimaEnv)
	env.profile = "local"

	if processName == "" {
		processName = determineProcessName()
	}

	env.systemProc = NewMockSystemProc(processName)
	env.folderGuide = NewMockFolderGuide(env.systemProc)

	return env
}

func (env *MockFatimaEnv) GetSystemProc() fatima.SystemProc {
	return env.systemProc
}

func (env *MockFatimaEnv) GetFolderGuide() fatima.FolderGuide {
	return env.folderGuide
}

func (env *MockFatimaEnv) GetProfile() string {
	return env.profile
}

func (env *MockFatimaEnv) SetProfile(profile string) {
	env.profile = profile
}

// MockSystemProc
type MockSystemProc struct {
	pid         int
	uid         int
	gid         string
	programName string
	username    string
	homeDir     string
}

func NewMockSystemProc(programName string) *MockSystemProc {
	proc := new(MockSystemProc)
	proc.pid = os.Getpid()
	proc.uid = os.Getuid()
	proc.programName = programName

	// get username and homedir
	// simple implementation
	proc.username = "mockuser"
	proc.homeDir = os.TempDir()
	proc.gid = "mockgroup"

	return proc
}

func (p *MockSystemProc) GetPid() int {
	return p.pid
}

func (p *MockSystemProc) GetUid() int {
	return p.uid
}

func (p *MockSystemProc) GetProgramName() string {
	return p.programName
}

func (p *MockSystemProc) GetUsername() string {
	return p.username
}

func (p *MockSystemProc) GetHomeDir() string {
	return p.homeDir
}

func (p *MockSystemProc) GetGid() string {
	return p.gid
}

// MockFolderGuide
type MockFolderGuide struct {
	fatimaHomePath string
	app            string
	bin            string
	conf           string
	data           string
	javalib        string
	lib            string
	log            string
	pack           string
	stat           string
	proc           string
}

func NewMockFolderGuide(proc fatima.SystemProc) *MockFolderGuide {
	guide := new(MockFolderGuide)

	fatimaHome := os.Getenv(fatima.ENV_FATIMA_HOME)
	if fatimaHome == "" {
		fatimaHome = filepath.Join(os.TempDir(), "fatima_home")
		_ = os.MkdirAll(fatimaHome, 0755)
	}
	guide.fatimaHomePath = fatimaHome

	guide.resolveFolder(proc.GetProgramName())

	return guide
}

func (this *MockFolderGuide) GetFatimaHome() string {
	return this.fatimaHomePath
}

func (this *MockFolderGuide) GetPackageProcFile() string {
	return fmt.Sprintf("%s%c%s", this.conf, os.PathSeparator, builder.FatimaFileProcConfig)
}

func (this *MockFolderGuide) GetAppProcFolder() string {
	return this.proc
}

func (this *MockFolderGuide) GetLogFolder() string {
	return this.log
}

func (this *MockFolderGuide) GetConfFolder() string {
	return this.conf
}

func (this *MockFolderGuide) IsAppExist() bool {
	if strings.HasSuffix(this.app, builder.GoTestProgramSuffix) {
		return false
	}

	if _, err := os.Stat(this.app); err == nil {
		return true
	}
	return false
}

func (this *MockFolderGuide) GetDataFolder() string {
	return this.data
}

func (this *MockFolderGuide) GetAppFolder() string {
	return this.app
}

func (this *MockFolderGuide) CreateTmpFolder() string {
	seed := lib.RandomAlphanumeric(16)
	tmp := filepath.Join(this.data, ".tmp", seed)
	checkDirectory(tmp, true)
	return tmp
}

func (this *MockFolderGuide) CreateTmpFilePath() string {
	seed := lib.RandomAlphanumeric(16)
	tmpDir := filepath.Join(this.data, ".tmp", seed)
	checkDirectory(tmpDir, true)
	seed = lib.RandomAlphanumeric(16)
	return filepath.Join(tmpDir, seed)
}

func (this *MockFolderGuide) resolveFolder(programName string) {
	this.app = filepath.Join(this.fatimaHomePath, builder.FatimaFolderApp, programName)
	checkDirectory(this.app, true)

	this.bin = filepath.Join(this.fatimaHomePath, builder.FatimaFolderBin, programName)
	checkDirectory(this.bin, false)

	this.conf = filepath.Join(this.fatimaHomePath, builder.FatimaFolderConf)
	checkDirectory(this.conf, false)

	this.data = filepath.Join(this.fatimaHomePath, builder.FatimaFolderData, programName)
	checkDirectory(this.data, true)

	this.javalib = filepath.Join(this.fatimaHomePath, builder.FatimaFolderJavalib)
	checkDirectory(this.javalib, false)

	this.lib = filepath.Join(this.fatimaHomePath, builder.FatimaFolderLib)
	checkDirectory(this.lib, false)

	this.log = filepath.Join(this.fatimaHomePath, builder.FatimaFolderLog, programName)
	checkDirectory(this.log, true)

	this.pack = filepath.Join(this.fatimaHomePath, builder.FatimaFolderPackage)
	checkDirectory(this.pack, false)

	this.stat = filepath.Join(this.fatimaHomePath, builder.FatimaFolderStat, programName)
	checkDirectory(this.stat, true)

	this.proc = filepath.Join(this.app, builder.FatimaFolderProc)
	checkDirectory(this.proc, true)

	_ = os.RemoveAll(filepath.Join(this.data, ".tmp"))
}

func checkDirectory(path string, forceCreate bool) {
	if err := ensureDirectory(path, forceCreate); err != nil {
		panic(err.Error())
	}
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

// Helper functions for process name determination

func determineProcessName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}

	root, found := findProjectRoot(wd)
	if !found {
		return filepath.Base(wd)
	}

	// check cmd folder
	cmdDir := filepath.Join(root, "cmd")
	if stat, err := os.Stat(cmdDir); err == nil && stat.IsDir() {
		entries, err := os.ReadDir(cmdDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					// check main.go
					mainFile := filepath.Join(cmdDir, entry.Name(), "main.go")
					if _, err := os.Stat(mainFile); err == nil {
						return entry.Name()
					}

					// check {dir_name}.go
					altMainFile := filepath.Join(cmdDir, entry.Name(), entry.Name()+".go")
					if _, err := os.Stat(altMainFile); err == nil {
						return entry.Name()
					}
				}
			}
		}
	}

	// fallback to project name (from go.mod or folder name)
	return filepath.Base(root)
}

func findProjectRoot(path string) (string, bool) {
	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			return path, true
		}

		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return "", false
}
