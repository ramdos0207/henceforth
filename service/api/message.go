package api

import (
	"context"
	"log"

	"github.com/google/uuid"
	traq "github.com/traPtitech/go-traq"
)

/*
// メッセージ投稿リクエストボディ
type Message struct {
	Content string `json:"content,omitempty"`
	Embed   bool   `json:"embed,omitempty"`
}
type MessageRes struct {
	// メッセージUUID
	Id string `json:"id"`
}
*/
// 指定されたチャンネルに指定されたメッセージを投稿してメッセージの UUID を返す
func (api *API) SendMessage(chanID string, message string) (string, error) {
	// 開発モードではコンソールにメッセージを表示するのみ
	if api.config.Dev_Mode {
		log.Printf("Sending\n%s\nto %s", message, chanID)
		id, _ := uuid.NewRandom()
		return id.String(), nil // ダミーの UUID を返す
	} else {
		client := traq.NewAPIClient(traq.NewConfiguration())
		auth := context.WithValue(context.Background(), traq.ContextAccessToken, api.config.Bot_Access_Token)

		embed := false
		v, _, err := client.MessageApi.PostMessage(auth, chanID).PostMessageRequest(traq.PostMessageRequest{Content: message, Embed: &embed}).Execute()
		if err != nil {
			return "", err
		}
		return v.Id, nil
	}
}

// デプロイ完了を config で設定したチャンネルに通知
func (api *API) NotifyDeployed() {
	api.SendMessage(api.config.Log_Chan_ID, "Log: The new version of Henceforth is deployed.")
}
