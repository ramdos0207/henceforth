package service

import (
	"fmt"

	"github.com/logica0419/scheduled-messenger-bot/service/api"
)

func SendCreateErrorMessage(api *api.API, channelID string, err error) {
	_ = api.SendMessage(channelID, fmt.Sprintf("メッセージの予約に失敗しました\n```plaintext\n%s\n```", err))
}

func SendDeleteErrorMessage(api *api.API, channelID string, err error) {
	_ = api.SendMessage(channelID, fmt.Sprintf("メッセージの削除に失敗しました\n```plaintext\n%s\n```", err))
}

func SendEditErrorMessage(api *api.API, channelID string, err error) {
	_ = api.SendMessage(channelID, fmt.Sprintf("メッセージの編集に失敗しました\n```plaintext\n%s\n```", err))
}
