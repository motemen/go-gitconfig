package gitconfig

func GetString(key string) (string, error)    { return Default.GetString(key) }
func GetStrings(key string) ([]string, error) { return Default.GetStrings(key) }
func GetPath(key string) (string, error)      { return Default.GetPath(key) }
func GetPaths(key string) ([]string, error)   { return Default.GetPaths(key) }
func GetBool(key string) (bool, error)        { return Default.GetBool(key) }
func GetInt64(key string) (int64, error)      { return Default.GetInt64(key) }
func Load(v interface{}) error                { return Default.Load(v) }
