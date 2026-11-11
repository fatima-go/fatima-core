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

package fatima

import (
	"errors"
	"fmt"
)

type Predefines interface {
	ResolvePredefine(value string) string
	GetDefine(key string) (string, bool)
}

type Config interface {
	GetValue(key string) (string, bool)
	GetString(key string) (string, error)
	GetInt(key string) (int, error)
	GetBool(key string) (bool, error)
	GetList(key string) ([]string, error)
}

type Packaging interface {
	GetName() string
	GetHost() string
	GetGroup() string
}

// StructuredConfig is an optional extension of Config. Paths contain literal map
// keys, not dot-separated expressions. An empty path selects the entire root.
// LookupNode returns a detached copy; (nil, true) denotes explicit null.
// DecodeNode requires a non-nil pointer and never coerces strings into numbers.
type StructuredConfig interface {
	LookupNode(path ...string) (any, bool)
	DecodeNode(dst any, path ...string) error
}

// ConfigLoadStatus is an optional extension of Config for startup validation.
// LoadError reports file read/parse failures, or ErrConfigNotFound if no config
// was found. It does not validate resolved secrets, variables or business rules.
// HasBaseConfig reports whether the selected base file loaded successfully.
// Profile-only configurations are valid; consumers may require a base explicitly.
type ConfigLoadStatus interface {
	LoadError() error
	HasBaseConfig() bool
}

var (
	ErrConfigNotFound   = errors.New("application config not found")
	ErrConfigLoadFailed = errors.New("application config file could not be loaded")
)

// ConfigLoadError identifies a failed file by basename without exposing its
// contents. Err is ErrConfigLoadFailed; original parser errors are not retained.
type ConfigLoadError struct {
	File string
	Err  error
}

func (e *ConfigLoadError) Error() string {
	return fmt.Sprintf("config file %q: %s", e.File, e.Err)
}
func (e *ConfigLoadError) Unwrap() error { return e.Err }

var (
	ErrConfigNodeNotFound  = errors.New("config node not found")
	ErrConfigTypeMismatch  = errors.New("config node type mismatch")
	ErrInvalidConfigTarget = errors.New("invalid config decode target")
)

// ConfigNodeError identifies a path and expected type without exposing values.
type ConfigNodeError struct {
	Path     []string
	Expected string
	Err      error
}

func (e *ConfigNodeError) Error() string {
	return fmt.Sprintf("config path %q: %s (expected %s)", e.Path, e.Err, e.Expected)
}
func (e *ConfigNodeError) Unwrap() error { return e.Err }
