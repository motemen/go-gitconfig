package gitconfig_test

import (
	"fmt"
	"github.com/motemen/go-gitconfig"
)

type config struct {
	UserEmail  string `gitconfig:"user.email"`
	PullRebase bool   `gitconfig:"pull.rebase"`
	GCAuto     int    `gitconfig:"gc.auto"`
}

func ExampleConfig_Load() {
	var v config
	gitconfig.Default.Load(&v)

	fmt.Println(v)
	// Output: {local@example.com true 6700}
}
