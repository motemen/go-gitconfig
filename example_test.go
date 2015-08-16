package gitconfig_test

import (
	"github.com/motemen/go-gitconfig"
)

func ExampleLoad() {
	type config struct {
		UserEmail  string `gitconfig:"user.email"`
		PullRebase bool   `gitconfig:"pull.rebase"`
		GCAuto     int    `gitconfig:"gc.auto"`
	}

	var v config
	gitconfig.Load(&v)
}
