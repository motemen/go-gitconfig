# go-gitconfig

Go library for loading git config

[GoDoc](http://godoc.org/github.com/motemen/go-gitconfig)

## Synopsis

### Struct loading

```go
import "github.com/motemen/go-gitconfig"

type C struct {
    UserEmail  string `gitconfig:"user.email"`
    PullRebase bool   `gitconfig:"pull.rebase"`
}

func main() {
    var config C
    err := gitconfig.Load(&config)
    ...
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
