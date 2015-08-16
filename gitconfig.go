// Package gitconfig is an interface to "git config" for reading values.
//
// The GetXXX methods return values of their corresponding types,
// or errors if "git config" fails e.g. when the key was invalid.
//
// Use Default (or gitconfig package itself), Global, Local, File(file) and Blob(blob) as an entrypoint
// to a specific config source. They correspond to flags below:
//   Default    (none)
//   Global     --global
//   Local      --local
//   File(file) --file <file>
//   Blob(blob) --blob <blob>
//
// Use Load method for loading multiple config values to a struct with fields tagged "gitconfig".
//   type Config struct {
//     UserEmail  string `gitconfig:"user.email"`
//     PullRebase bool   `gitconfig:"pull.rebase"`
//   }
// Supported types are string, []string, bool and int families.
package gitconfig

import (
	"bytes"
	"fmt"
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

// IsInvalidKeyError returns true if the given err is a RunError
// corresponding to "invalid key" error of "git config".
func IsInvalidKeyError(err error) bool {
	if err, ok := err.(RunError); ok {
		if waitStatus, ok := err.Err.Sys().(syscall.WaitStatus); ok {
			return waitStatus.ExitStatus() == 1
		}
	}

	return false
}

// RunError is a general error for "git config" failure.
type RunError struct {
	Msg string
	Err *exec.ExitError
}

func (err RunError) Error() string {
	return err.Err.Error() + ": " + err.Msg
}

// LoadError is an error type for Load().
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

// OfField returns the error corresponding to a given field.
func (m LoadError) OfField(name string) error {
	return m[name]
}

// Any returns nil if there was actually no error, and itself otherwise.
func (m LoadError) Any() LoadError {
	if len(m) == 0 {
		return nil
	}

	return m
}

func (c Config) get(key string, extraArgs ...string) ([]string, error) {
	args := []string{"config", "--get-all", "--null"}
	args = append(args, c.Source...)
	args = append(args, extraArgs...)
	args = append(args, key)

	var stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr

	out, err := cmd.Output()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, RunError{
				Msg: strings.TrimRight(stderr.String(), "\n"),
				Err: exitError,
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
