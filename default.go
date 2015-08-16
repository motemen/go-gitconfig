package gitconfig

// GetString is a shortcut for Default.GetString.
func GetString(key string) (string, error) { return Default.GetString(key) }

// GetStrings is a shortcut for Default.GetStrings.
func GetStrings(key string) ([]string, error) { return Default.GetStrings(key) }

// GetPath is a shortcut for Default.Path.
func GetPath(key string) (string, error) { return Default.GetPath(key) }

// GetPaths is a shortcut for Default.Paths.
func GetPaths(key string) ([]string, error) { return Default.GetPaths(key) }

// GetBool is a shortcut for Default.Bool.
func GetBool(key string) (bool, error) { return Default.GetBool(key) }

// GetInt64 is a shortcut for Default.Int64.
func GetInt64(key string) (int64, error) { return Default.GetInt64(key) }

// Load is a shortcut for Default.Load.
func Load(v interface{}) error { return Default.Load(v) }
