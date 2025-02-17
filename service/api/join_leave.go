package api

import (
	"context"

	traq "github.com/traPtitech/go-traq"
)

// JOIN / LEAVE のリクエストボディ
type ActionBody struct {
	ChannelID string `json:"channelId,omitempty"`
}

// 指定されたチャンネルに JOIN / LEAVE する
func (api *API) JoinChannel(cmd string, chanID string) error {
	client := traq.NewAPIClient(traq.NewConfiguration())
	auth := context.WithValue(context.Background(), traq.ContextAccessToken, api.config.Bot_Access_Token)

	_, err := client.BotApi.LetBotJoinChannel(auth, api.config.Bot_ID).PostBotActionJoinRequest(traq.PostBotActionJoinRequest{ChannelId: chanID}).Execute()

	return err
}
func (api *API) LeaveChannel(cmd string, chanID string) error {
	client := traq.NewAPIClient(traq.NewConfiguration())
	auth := context.WithValue(context.Background(), traq.ContextAccessToken, api.config.Bot_Access_Token)

	_, err := client.BotApi.LetBotLeaveChannel(auth, api.config.Bot_ID).PostBotActionLeaveRequest(traq.PostBotActionLeaveRequest{ChannelId: chanID}).Execute()

	return err
}
