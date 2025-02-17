package api

import (
	"github.com/logica0419/scheduled-messenger-bot/config"
)

type API struct {
	Config *config.Config
}

func GetApi(c *config.Config) *API {
	api := &API{Config: c}
	return api
}
