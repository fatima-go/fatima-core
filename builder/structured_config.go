package builder

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	fatima "github.com/fatima-go/fatima-core"
)

// ConfigSnapshot is an immutable view of resolved configuration. Its zero value
// represents an unavailable configuration. Lookups and decoding return copies.
type ConfigSnapshot struct {
	root       map[string]any
	loadErrors []fatima.ConfigLoadError
	baseLoaded bool
}

var _ fatima.StructuredConfig = (*ConfigSnapshot)(nil)
var _ fatima.StructuredConfig = (*PropertyConfigReader)(nil)
var _ fatima.ConfigLoadStatus = (*ConfigSnapshot)(nil)
var _ fatima.ConfigLoadStatus = (*PropertyConfigReader)(nil)

// LoadError returns detached diagnostics so callers cannot alter load status.
// A partial snapshot remains readable even when loading reported errors.
func (s *ConfigSnapshot) LoadError() error {
	if s == nil || s.root == nil {
		return fatima.ErrConfigNotFound
	}
	errs := make([]error, len(s.loadErrors))
	for i, diagnostic := range s.loadErrors {
		copy := diagnostic
		errs[i] = &copy
	}
	return errors.Join(errs...)
}

func (s *ConfigSnapshot) HasBaseConfig() bool {
	return s != nil && s.baseLoaded
}

// NewStringConfigSnapshot copies flat, already resolved string settings.
// Keys remain literal; no YAML structure or value types are inferred.
func NewStringConfigSnapshot(values map[string]string) *ConfigSnapshot {
	root := make(map[string]any, len(values))
	for k, v := range values {
		root[k] = v
	}
	return &ConfigSnapshot{root: root}
}

func cloneConfigNode(value any) any {
	switch n := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(n))
		for k, v := range n {
			result[k] = cloneConfigNode(v)
		}
		return result
	case []any:
		result := make([]any, len(n))
		for i, v := range n {
			result[i] = cloneConfigNode(v)
		}
		return result
	default:
		return value
	}
}

func (s *ConfigSnapshot) lookup(path []string) (any, bool) {
	if s == nil || s.root == nil {
		return nil, false
	}
	var value any = s.root
	for _, key := range path {
		m, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return value, true
}

func (s *ConfigSnapshot) LookupNode(path ...string) (any, bool) {
	value, ok := s.lookup(path)
	return cloneConfigNode(value), ok
}

func configDecodeError(path []string, expected string, cause error) error {
	return &fatima.ConfigNodeError{Path: append([]string(nil), path...), Expected: expected, Err: cause}
}

// DecodeNode decodes into a fresh value and assigns dst only on success. Struct
// fields use yaml tags (or lower-case field names); unknown input keys are ignored.
// Supported destinations are scalars, pointers, structs, slices, arrays,
// string-keyed maps and empty interfaces. Custom unmarshalers are not invoked.
func (s *ConfigSnapshot) DecodeNode(dst any, path ...string) error {
	target := reflect.ValueOf(dst)
	if !target.IsValid() || target.Kind() != reflect.Pointer || target.IsNil() {
		return configDecodeError(path, "non-nil pointer", fatima.ErrInvalidConfigTarget)
	}
	value, ok := s.lookup(path)
	if !ok {
		return configDecodeError(path, target.Type().Elem().String(), fatima.ErrConfigNodeNotFound)
	}
	temporary := reflect.New(target.Type().Elem()).Elem()
	if err := decodeConfigNode(value, temporary, path); err != nil {
		return err
	}
	target.Elem().Set(temporary)
	return nil
}

func decodeConfigNode(value any, dst reflect.Value, path []string) error {
	mismatch := func() error { return configDecodeError(path, dst.Type().String(), fatima.ErrConfigTypeMismatch) }
	if value == nil {
		switch dst.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			dst.SetZero()
			return nil
		default:
			return mismatch()
		}
	}
	if dst.Kind() == reflect.Pointer {
		dst.Set(reflect.New(dst.Type().Elem()))
		return decodeConfigNode(value, dst.Elem(), path)
	}
	src := reflect.ValueOf(value)
	switch dst.Kind() {
	case reflect.Interface:
		if dst.NumMethod() != 0 {
			return mismatch()
		}
		dst.Set(reflect.ValueOf(cloneConfigNode(value)))
	case reflect.String:
		if src.Kind() != reflect.String {
			return mismatch()
		}
		dst.SetString(src.String())
	case reflect.Bool:
		if src.Kind() != reflect.Bool {
			return mismatch()
		}
		dst.SetBool(src.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var n int64
		switch src.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n = src.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if src.Uint() > math.MaxInt64 {
				return mismatch()
			}
			n = int64(src.Uint())
		default:
			return mismatch()
		}
		if dst.OverflowInt(n) {
			return mismatch()
		}
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var n uint64
		switch src.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if src.Int() < 0 {
				return mismatch()
			}
			n = uint64(src.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n = src.Uint()
		default:
			return mismatch()
		}
		if dst.OverflowUint(n) {
			return mismatch()
		}
		dst.SetUint(n)
	case reflect.Float32, reflect.Float64:
		if src.Kind() != reflect.Float32 && src.Kind() != reflect.Float64 {
			return mismatch()
		}
		if dst.OverflowFloat(src.Float()) {
			return mismatch()
		}
		dst.SetFloat(src.Float())
	case reflect.Slice, reflect.Array:
		list, ok := value.([]any)
		if !ok {
			return mismatch()
		}
		if dst.Kind() == reflect.Array {
			if dst.Len() != len(list) {
				return mismatch()
			}
		} else {
			dst.Set(reflect.MakeSlice(dst.Type(), len(list), len(list)))
		}
		for i, v := range list {
			next := append(append([]string(nil), path...), fmt.Sprintf("[%d]", i))
			if err := decodeConfigNode(v, dst.Index(i), next); err != nil {
				return err
			}
		}
	case reflect.Map:
		m, ok := value.(map[string]any)
		if !ok || dst.Type().Key().Kind() != reflect.String {
			return mismatch()
		}
		dst.Set(reflect.MakeMapWithSize(dst.Type(), len(m)))
		for _, k := range sortedConfigKeys(m) {
			elem := reflect.New(dst.Type().Elem()).Elem()
			next := append(append([]string(nil), path...), k)
			if err := decodeConfigNode(m[k], elem, next); err != nil {
				return err
			}
			key := reflect.New(dst.Type().Key()).Elem()
			key.SetString(k)
			dst.SetMapIndex(key, elem)
		}
	case reflect.Struct:
		// yaml.v3 timestamps are time.Time values, which can be copied directly.
		if src.Type() == dst.Type() {
			dst.Set(src)
			return nil
		}
		m, ok := value.(map[string]any)
		if !ok {
			return mismatch()
		}
		seen := make(map[string]bool)
		for i := 0; i < dst.NumField(); i++ {
			field := dst.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			tag := strings.Split(field.Tag.Get("yaml"), ",")
			name := tag[0]
			if name == "-" {
				continue
			}
			for _, option := range tag[1:] {
				if option != "omitempty" && option != "flow" && option != "" {
					return configDecodeError(path, dst.Type().String()+" with named yaml fields (no inline)", fatima.ErrInvalidConfigTarget)
				}
			}
			if name == "" {
				name = strings.ToLower(field.Name)
			}
			if seen[name] {
				return configDecodeError(path, "unique yaml field names", fatima.ErrInvalidConfigTarget)
			}
			seen[name] = true
			if v, exists := m[name]; exists {
				next := append(append([]string(nil), path...), name)
				if err := decodeConfigNode(v, dst.Field(i), next); err != nil {
					return err
				}
			}
		}
	default:
		return mismatch()
	}
	return nil
}
