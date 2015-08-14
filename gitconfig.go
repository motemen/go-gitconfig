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
	// SourceDefault is a default source (local and global).
	SourceDefault Source
	// SourceGlobal looks into global git config (e.g. ~/.gitconfig).
	SourceGlobal = []string{"--global"}
	// SourceLocal looks into local git config (e.g. .git/config).
	SourceLocal = []string{"--local"}
)

// SourceFile is a source that looks into a specified file.
func SourceFile(file string) Source {
	return Source{"--file", file}
}

func SourceBlob(blob string) Source {
	return Source{"--blob", blob}
}

// Config is the main interface of gitconfig package.
type Config struct {
	Source Source
}

var (
	// Default reads git config from default source e.g. local and global.
	Default = Config{}
	// Global reads git config from global source (e.g. ~/.gitconfig).
	Global = Config{Source: SourceGlobal}
	// Local reads git config from local source (e.g. .git/config).
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

type ErrInvalidKey string

func (err ErrInvalidKey) Error() string {
	return "invalid key: " + string(err)
}

type Errors []error

func (err Errors) Error() string {
	if len(err) == 0 {
		return "no error"
	} else if len(err) == 1 {
		return err[0].Error()
	} else {
		return fmt.Sprintf("%s and %d error(s)", err[0].Error(), len(err)-1)
	}
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
				return nil, ErrInvalidKey(key)
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

// GetPath obtains one path value. e.g. "~" expands to home directory.
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

	var errs Errors
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
				errs = append(errs, err)
				continue
			}
			fv.SetString(s)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := c.GetInt64(key)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			fv.SetInt(i)

		case reflect.Slice:
			ss, err := c.GetStrings(key)
			if err != nil {
				errs = append(errs, err)
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
				errs = append(errs, err)
				continue
			}

			for i := 0; i < fv.Len() && i < len(ss); i++ {
				fv.Index(i).SetString(ss[i])
			}

		case reflect.Bool:
			b, err := c.GetBool(key)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			fv.SetBool(b)

		default:
			err := fmt.Errorf("cannot populate field %q of type %s", ft.Name, ft.Type.String())
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
