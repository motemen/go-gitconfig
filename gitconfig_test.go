package gitconfig

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	panicIf := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	must := panicIf

	run := func(cmd string, args ...string) error {
		return exec.Command(cmd, args...).Run()
	}

	tmpHome, err := ioutil.TempDir("", "go-gitconfig")
	panicIf(err)

	repoDir := filepath.Join(tmpHome, "repo")
	must(os.MkdirAll(repoDir, 0777))

	must(os.Chdir(repoDir))

	must(os.Setenv("HOME", tmpHome))

	must(run("git", "init"))
	must(run("git", "config", "--global", "user.name", "go-gitconfig test"))
	must(run("git", "config", "--global", "user.email", "global@example.com"))
	must(run("git", "config", "--global", "merge.ff", "false"))
	must(run("git", "config", "--global", "pull.rebase", "true"))
	must(run("git", "config", "--global", "gc.auto", "6700"))
	must(run("git", "config", "--global", "--add", "ghq.root", "~/dev"))
	must(run("git", "config", "--global", "--add", "ghq.root", "~/go/src"))

	must(run("git", "config", "--local", "user.email", "local@example.com"))
	must(run("git", "config", "--local", "remote.origin.url", "git@example.com:repo.git"))

	must(ioutil.WriteFile(".gitmodules", []byte(`
[submodule "modules/sub"]
path = modules/sub
url = https://git.example.com/sub.git
`), 0777))

	must(run("git", "add", "."))
	must(run("git", "commit", "-m", "initial commit"))

	os.Exit(m.Run())
}

func TestShowEnv(t *testing.T) {
	t.Logf("HOME=%s", os.Getenv("HOME"))

	wd, _ := os.Getwd()
	t.Logf("WORK=%s", wd)
}

func TestGetString(t *testing.T) {
	assert := assert.New(t)

	paths, err := GetStrings("ghq.root")
	assert.NoError(err)
	assert.Equal([]string{
		"~/dev",
		"~/go/src",
	}, paths)

	path, err := GetString("ghq.root")
	assert.NoError(err)
	assert.Equal(paths[1], path)
}

func TestGetPaths(t *testing.T) {
	assert := assert.New(t)

	paths, err := GetPaths("ghq.root")
	assert.NoError(err)
	assert.Equal([]string{
		filepath.Join(os.Getenv("HOME"), "dev"),
		filepath.Join(os.Getenv("HOME"), "go", "src"),
	}, paths)

	path, err := GetPath("ghq.root")
	assert.NoError(err)
	assert.Equal(paths[1], path)
}

func TestFile(t *testing.T) {
	assert := assert.New(t)

	url, err := File(".gitmodules").GetString("submodule.modules/sub.url")
	assert.NoError(err)
	assert.Equal("https://git.example.com/sub.git", url)

	_, err = File("nonexistent").GetString("submodule.modules/sub.url")
	assert.Error(err)
}

func TestBlob(t *testing.T) {
	assert := assert.New(t)

	url, err := Blob("HEAD:.gitmodules").GetString("submodule.modules/sub.url")
	assert.NoError(err)
	assert.Equal("https://git.example.com/sub.git", url)

	_, err = Blob("nonexistent").GetString("submodule.modules/sub.url")
	assert.Error(err)
}

func TestGetInt64(t *testing.T) {
	assert := assert.New(t)

	i, err := GetInt64("gc.auto")
	assert.NoError(err)
	assert.Equal(int64(6700), i)
}

func TestGetBool(t *testing.T) {
	assert := assert.New(t)

	{
		b, err := GetBool("merge.ff")
		assert.NoError(err)
		assert.Equal(false, b)
	}

	{
		b, err := GetBool("pull.rebase")
		assert.NoError(err)
		assert.Equal(true, b)
	}
}

func TestSources(t *testing.T) {
	assert := assert.New(t)

	d, err := Default.GetString("remote.origin.url")
	assert.NoError(err)
	assert.Equal("git@example.com:repo.git", d)

	l, err := Local.GetString("remote.origin.url")
	assert.NoError(err)
	assert.Equal("git@example.com:repo.git", l)

	g, err := Global.GetString("remote.origin.url")
	assert.NotNil(err)
	assert.Equal("", g)

	f, err := File(".gitmodules").GetString("remote.origin.url")
	assert.NotNil(err)
	assert.Equal("", f)

	b, err := Blob("HEAD:.gitmodules").GetString("remote.origin.url")
	assert.NotNil(err)
	assert.Equal("", b)
}

func TestLoad(t *testing.T) {
	assert := assert.New(t)

	type s struct {
		UserEmail  string   `gitconfig:"user.email"`
		GCAuto     int      `gitconfig:"gc.auto"`
		PullRebase bool     `gitconfig:"pull.rebase"`
		GhqRoots   []string `gitconfig:"ghq.root"`
	}

	var v s
	err := Load(&v)
	assert.Nil(err)
	assert.Equal(
		s{
			UserEmail:  "local@example.com",
			GCAuto:     6700,
			PullRebase: true,
			GhqRoots: []string{
				"~/dev", "~/go/src", // TODO expand as path
			},
		},
		v,
	)
}

func TestLoad_Errors(t *testing.T) {
	assert := assert.New(t)

	type s struct {
		UserEmail    string `gitconfig:"user.email"`
		InvalidKey   string `gitconfig:"invalidkey"`
		InvalidType  bool   `gitconfig:"user.email"`
		InvalidType2 *s     `gitconfig:"gc.auto"`
	}

	var v s
	err := Load(&v)
	assert.Error(err)

	if m, ok := err.(LoadError); assert.True(ok) {
		assert.Error(m.OfField("InvalidKey"))
		assert.Error(m.OfField("InvalidType"))
		assert.Error(m.OfField("InvalidType2"))
		assert.Nil(m.OfField("UserEmail"))

		assert.True(IsInvalidKeyError(m.OfField("InvalidKey")))
		assert.False(IsInvalidKeyError(m.OfField("InvalidType")))
		assert.False(IsInvalidKeyError(m.OfField("InvalidType2")))
	}
}
