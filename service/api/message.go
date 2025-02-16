package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// メッセージ投稿リクエストボディ
type Message struct {
	Content string `json:"content,omitempty"`
	Embed   bool   `json:"embed,omitempty"`
}
type MessageRes struct {
	// メッセージUUID
	Id string `json:"id"`
}

// 指定されたチャンネルに指定されたメッセージを投稿してメッセージの UUID を返す
func (api *API) SendMessage(chanID string, message string) (string, error) {
	// 開発モードではコンソールにメッセージを表示するのみ
	if api.config.Dev_Mode {
		log.Printf("Sending\n%s\nto %s", message, chanID)
		id, _ := uuid.NewRandom()
		return id.String(), nil // ダミーの UUID を返す
	} else {
		// URL を生成
		url := fmt.Sprintf("%s/channels/%s/messages", baseUrl, chanID)

		// ボディを作成
		body := Message{Content: message, Embed: false}

		// リクエストを送信
		res, err := api.post(url, body)
		if err != nil {
			return "", err
		}
		var mesRes MessageRes
		if err := json.Unmarshal(res, &mesRes); err != nil {
			return "", err
		}
		return mesRes.Id, nil
	}
}

// デプロイ完了を config で設定したチャンネルに通知
func (api *API) NotifyDeployed() {
	api.SendMessage(api.config.Log_Chan_ID, "Log: The new version of Henceforth is deployed.")
}
