package handler

import "honoka-chan/config"

// ReloadConfigGlobals refreshes handler globals that were copied from config.Conf
// during init(). Keep this in sync with existing global variables only; do not
// change the config.json schema.
func ReloadConfigGlobals() {
    SifCdnServer = config.Conf.Settings.SifCdnServer
    AsCdnServer = config.Conf.Settings.AsCdnServer
}
