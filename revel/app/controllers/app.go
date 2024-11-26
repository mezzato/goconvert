package controllers

import (
	"encoding/json"
	"path/filepath"

	"github.com/mezzato/goconvert/settings"
	"github.com/revel/revel"
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

	p := &Page{WebPort: revel.HTTPPort, SettingsAsJson: string(settingsAsJson)}

	return c.Render(p)
}
