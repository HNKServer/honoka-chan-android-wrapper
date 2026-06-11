package config

import (
    "encoding/json"
    "honoka-chan/utils"
)

// ParseConfigFileStrict parses the existing config.json without changing the file.
// Unlike Load(), it does not rename a bad config or regenerate defaults. This is
// important for the Android UI editor: invalid JSON should simply be rejected.
func ParseConfigFileStrict(p string) (*AppConfigs, error) {
    c := AppConfigs{}
    data := utils.ReadAllText(p)
    if err := json.Unmarshal([]byte(data), &c); err != nil {
        return nil, err
    }
    return &c, nil
}

// ReloadConfigFileStrict reloads the existing config.json into runtime memory.
// It preserves the original config.json schema completely.
func ReloadConfigFileStrict(p string) error {
    c, err := ParseConfigFileStrict(p)
    if err != nil {
        return err
    }
    Conf = c
    return nil
}
