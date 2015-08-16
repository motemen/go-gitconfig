# go-gitconfig

Go library for loading git config

## Synopsis

### Struct loading

```go
import "github.com/motemen/go-gitconfig"

type C struct {
    UserEmail  string `gitconfig:"user.email"`
    PullRebase string `gitconfig:"pull.rebase"`
}

func main() {
    var config C
    gitconfig.Load(&config)
}
```

### Or simply access the key

```go
url, err := gitconfig.GetString("remote.origin.url")
```

```go
n, err := gitconfig.GetInt64("gc.auto")
```

### Specifying sources

Use below to change git config sources:

- `gitconfig.Global`
- `gitconfig.Local`
- `gitconfig.File(file)`
- `gitconfig.Blob(blob)`

## Author

motemen <motemen@gmail.com>
