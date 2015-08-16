// Package gitconfig is an interface to `git config` for reading values.
package gitconfig

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

// Source is a source (global, local, file...) of git config.
type Source []string

var (
	// SourceDefault is a default source (local, global and system).
	SourceDefault Source
	// SourceGlobal looks into global git config (eg. ~/.gitconfig).
	SourceGlobal = []string{"--global"}
	// SourceLocal looks into local git config (eg. .git/config).
	SourceLocal = []string{"--local"}
)

// SourceFile is a source that looks into a specified file.
func SourceFile(file string) Source {
	return Source{"--file", file}
}

// SourceBlob is a source that looks into a specified blob (eg. HEAD:.gitmodules)
func SourceBlob(blob string) Source {
	return Source{"--blob", blob}
}

// Config is the main interface of gitconfig package.
type Config struct {
	Source Source
}

var (
	// Default reads git config from default source i.e. local, global and system
	Default = Config{}
	// Global reads git config from global source (eg. ~/.gitconfig).
	Global = Config{Source: SourceGlobal}
	// Local reads git config from local source (eg. .git/config).
	Local = Config{Source: SourceLocal}
)

// File reads git config from specified file.
func File(file string) Config {
	return Config{Source: SourceFile(file)}
}

// Blob reads git config from specified blob.
func Blob(blob string) Config {
	return Config{Source: SourceBlob(blob)}
}

// InvalidKeyError represents an error for an invalid config key.
type InvalidKeyError string

func (err InvalidKeyError) Error() string {
	return "invalid key: " + string(err)
}

type LoadError map[string]error

func (m LoadError) Error() string {
	if len(m) == 0 {
		return "(no error)"
	}

	ee := make([]string, 0, len(m))
	for name, err := range m {
		ee = append(ee, fmt.Sprintf("field %q: %s", name, err))
	}

	return strings.Join(ee, "\n")
}

func (m LoadError) OfField(name string) error {
	return m[name]
}

func (m LoadError) Any() LoadError {
	if len(m) == 0 {
		return nil
	}

	return m
}

func (c Config) get(key string, extraArgs ...string) ([]string, error) {
	args := append([]string{"config", "--get-all", "--null"}, c.Source...)
	args = append(args, extraArgs...)
	args = append(args, key)

	cmd := exec.Command("git", args...)
	cmd.Stderr = ioutil.Discard

	out, err := cmd.Output()

	if exitError, ok := err.(*exec.ExitError); ok {
		if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
			if waitStatus.ExitStatus() == 1 {
				return nil, InvalidKeyError(key)
			}
		}
		return nil, err
	}

	ss := strings.Split(string(out), "\000")
	return ss[:len(ss)-1], nil
}

// GetString obtains one string value.
func (c Config) GetString(key string) (string, error) {
	values, err := c.get(key)
	if err != nil {
		return "", err
	}

	return values[len(values)-1], nil
}

// GetStrings obtains multiple string values.
func (c Config) GetStrings(key string) ([]string, error) {
	return c.get(key)
}

// GetPath obtains one path value. eg. "~" expands to home directory.
func (c Config) GetPath(key string) (string, error) {
	values, err := c.get(key, "--path")
	if err != nil {
		return "", err
	}

	return values[len(values)-1], nil
}

// GetPaths obtains multiple path values.
func (c Config) GetPaths(key string) ([]string, error) {
	return c.get(key, "--path")
}

// GetBool obtains one boolean value.
func (c Config) GetBool(key string) (bool, error) {
	values, err := c.get(key, "--bool")
	if err != nil {
		return false, err
	}

	return values[0] == "true", nil
}

// GetInt64 obtains one integer value.
func (c Config) GetInt64(key string) (int64, error) {
	values, err := c.get(key, "--int")
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(values[0], 10, 64)
}

// Load loads git config values to a struct annotated with "gitconfig" tags.
func (c Config) Load(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("not a pointer: %v", v)
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("not a pointer to a struct: %v", v)
	}

	t := rv.Type()

	errs := LoadError{}
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		fv := rv.Field(i)

		if fv.CanSet() == false {
			continue
		}

		tag := ft.Tag.Get("gitconfig")
		if tag == "" {
			continue
		}

		tags := strings.Split(tag, ",")

		var (
			key = tags[0]
			_   = tags[1:]
		)

		switch fv.Kind() {
		case reflect.String:
			s, err := c.GetString(key)
			if err != nil {
				errs[ft.Name] = err
				continue
			}
			fv.SetString(s)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := c.GetInt64(key)
			if err != nil {
				errs[ft.Name] = err
				continue
			}
			fv.SetInt(i)

		case reflect.Slice:
			ss, err := c.GetStrings(key)
			if err != nil {
				errs[ft.Name] = err
				continue
			}

			ssr := reflect.MakeSlice(reflect.TypeOf(ss), len(ss), len(ss))
			for i, s := range ss {
				ssr.Index(i).SetString(s)
			}

			fv.Set(ssr)

		case reflect.Array:
			ss, err := c.GetStrings(key)
			if err != nil {
				errs[ft.Name] = err
				continue
			}

			for i := 0; i < fv.Len() && i < len(ss); i++ {
				fv.Index(i).SetString(ss[i])
			}

		case reflect.Bool:
			b, err := c.GetBool(key)
			if err != nil {
				errs[ft.Name] = err
				continue
			}
			fv.SetBool(b)

		default:
			err := fmt.Errorf("cannot populate field %q of type %s", ft.Name, ft.Type.String())
			errs[ft.Name] = err
		}
	}

	return errs.Any()
}
