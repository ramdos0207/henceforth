package api

import (
	"github.com/logica0419/scheduled-messenger-bot/config"
)

type API struct {
	config *config.Config
}

func GetApi(c *config.Config) *API {
	api := &API{config: c}
	return api
}
