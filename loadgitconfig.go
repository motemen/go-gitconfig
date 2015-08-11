// Package loadgitconfig provides functions to populate values into structs
// from Git config files.
// "gitconfig" tag must be applied to exported fields of the struct:
//
//   type config struct {
//     Token  string `gitconfig:"my.token"`
//     Secure bool   `gitconfig:"my.secure"` // bool values are got using --bool
//     Max    int    `gitconfig:"my.max"`    // int values are got using --int
//   }
package loadgitconfig

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

type Source []string

var (
	SourceDefault Source = nil
	SourceGlobal         = []string{"--global"}
	SourceLocal          = []string{"--local"}
)

func SourceFile(file string) Source {
	return Source{"--file", file}
}

type Config struct {
	Source Source

	// extra arguments to "git config"
	CmdArgs []string
}

type ErrInvalidKey string

func (err ErrInvalidKey) Error() string {
	return "invalid key: " + string(err)
}

type Errors []error

func (err Errors) Error() string {
	return fmt.Sprintf("%d errors", len(err))
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

func (c Config) GetString(key string) (string, error) {
	values, err := c.get(key)
	if err != nil {
		return "", err
	}

	return values[len(values)-1], nil
}

func (c Config) GetStrings(key string) ([]string, error) {
	return c.get(key)
}

func (c Config) GetBool(key string) (bool, error) {
	values, err := c.get(key, "--bool")
	if err != nil {
		return false, err
	}

	return values[0] == "true", nil
}

func (c Config) GetInt64(key string) (int64, error) {
	values, err := c.get(key, "--int")
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(values[0], 10, 64)
}

func (c Config) Load(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("not a pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("not a pointer to a struct")
	}

	t := rv.Type()

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
				return err
			}
			fv.SetString(s)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := c.GetInt64(key)
			if err != nil {
				return err
			}
			fv.SetInt(i)

		case reflect.Slice:
			ss, err := c.GetStrings(key)
			if err != nil {
				return err
			}

			ssr := reflect.MakeSlice(reflect.TypeOf(ss), len(ss), len(ss))
			for i, s := range ss {
				ssr.Index(i).SetString(s)
			}

			fv.Set(ssr)

		case reflect.Array:
			ss, err := c.GetStrings(key)
			if err != nil {
				return err
			}

			for i := 0; i < fv.Len(); i++ {
				if i >= len(ss) {
					break
				}
				fv.Index(i).SetString(ss[i])
			}

		case reflect.Bool:
			b, err := c.GetBool(key)
			if err != nil {
				return err
			}
			fv.SetBool(b)
		}
	}

	return nil
}
