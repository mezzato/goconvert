package controllers

import (
	"github.com/mezzato/goconvert/settings"
	"encoding/json"
	"github.com/robfig/revel"
	"path/filepath"
)

type Page struct {
	Title          string
	WebPort        int
	SettingsAsJson string
}

var (
	homeImgDir                         = filepath.Join(settings.GetHomeDir(), "Pictures", "ToResize")
	defaultSettings *settings.Settings = settings.NewDefaultSettings("", homeImgDir)
)

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {

	settingsAsJson, err := json.Marshal(defaultSettings)
	if err != nil {
		return nil
	}

	p := &Page{WebPort: revel.HttpPort, SettingsAsJson: string(settingsAsJson)}

	return c.Render(p)
}
