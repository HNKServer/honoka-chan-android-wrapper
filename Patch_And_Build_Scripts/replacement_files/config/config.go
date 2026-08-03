package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	Conf = &AppConfigs{}

	PackageVersion = "97.4.6"

	PrivateKeyPath = "assets/certs/privatekey.pem"
	PublicKeyPath  = "assets/certs/publickey.pem"
)

type AppConfigs struct {
	AppName  string   `json:"app_name"`
	Settings Settings `json:"settings"`
}

type Settings struct {
	ListenPort               string `json:"listen_port"`
	CdnServer                string `json:"cdn_server"`
	UnlockAllSpecialRotation bool   `json:"unlock_all_special_rotation"`
}

func InitConfig() error {
	// Upgrade wrapper-era key names before the mainline loader reads the file.
	// Invalid JSON is left untouched so Load can keep the mainline backup/default behavior.
	if _, err := os.Stat("./config.json"); err == nil {
		if err := EnsureAndroidWrapperCompatibility("./config.json"); err != nil {
			log.Printf("Android wrapper 配置兼容迁移跳过: %v", err)
		}
	}

	conf, err := Load("./config.json")
	if err != nil {
		return err
	}
	Conf = conf
	return nil
}

func DefaultConfigs() *AppConfigs {
	return &AppConfigs{
		AppName: "honoka-chan",
		Settings: Settings{
			ListenPort:               "8080",
			CdnServer:                "http://127.0.0.1:8080/static",
			UnlockAllSpecialRotation: false,
		},
	}
}

func Load(p string) (*AppConfigs, error) {
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		conf := DefaultConfigs()
		if err := conf.Save(p); err != nil {
			return nil, fmt.Errorf("create default config: %w", err)
		}
		return conf, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	c := AppConfigs{}
	if err := json.Unmarshal(data, &c); err == nil {
		return &c, nil
	}

	backup := p + ".backup" + strconv.FormatInt(time.Now().Unix(), 10)
	if err := os.Rename(p, backup); err != nil {
		return nil, fmt.Errorf("backup invalid config: %w", err)
	}
	conf := DefaultConfigs()
	if err := conf.Save(p); err != nil {
		return nil, fmt.Errorf("write default config: %w", err)
	}
	return conf, nil
}

func (c *AppConfigs) Save(p string) error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0644)
}

// ReloadStrict replaces the in-memory configuration only when config.json is valid.
// Unlike Load, it never renames an invalid file and never creates a default file.
func ReloadStrict(path string) error {
	if err := EnsureAndroidWrapperCompatibility(path); err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var next AppConfigs
	if err := json.Unmarshal(data, &next); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	Conf = &next
	return nil
}

// EnsureAndroidWrapperCompatibility migrates only the old wrapper-era setting names.
// Unknown JSON fields are preserved so upgrading the Go core does not discard user data.
func EnsureAndroidWrapperCompatibility(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config for compatibility migration: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config for compatibility migration: %w", err)
	}

	settings, _ := raw["settings"].(map[string]any)
	if settings == nil {
		settings = map[string]any{}
		raw["settings"] = settings
	}

	changed := false
	if stringValue(settings["listen_port"]) == "" {
		port := stringValue(settings["server_port"])
		if port == "" {
			port = "8080"
		}
		settings["listen_port"] = port
		changed = true
	}

	if stringValue(settings["cdn_server"]) == "" {
		cdn := stringValue(settings["sif_cdn_server"])
		if cdn == "" {
			cdn = stringValue(settings["as_cdn_server"])
		}
		if cdn == "" {
			port := stringValue(settings["listen_port"])
			if port == "" {
				port = "8080"
			}
			cdn = "http://127.0.0.1:" + port + "/static"
		}
		settings["cdn_server"] = cdn
		changed = true
	}

	if !changed {
		return nil
	}

	updated, err := json.MarshalIndent(raw, "", "    ")
	if err != nil {
		return fmt.Errorf("encode migrated config: %w", err)
	}
	if err := os.WriteFile(path, append(updated, '\n'), 0644); err != nil {
		return fmt.Errorf("write migrated config: %w", err)
	}
	return nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
