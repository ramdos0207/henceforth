package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/logica0419/scheduled-messenger-bot/model"
	"github.com/logica0419/scheduled-messenger-bot/repository"
	"github.com/logica0419/scheduled-messenger-bot/service/api"
)

var ErrUserNotMatch = fmt.Errorf("access from different user")

// 予約投稿メッセージを DB から取得
func GetSchMesByID(repo repository.Repository, mesID string) (*model.SchMes, error) {
	// ID を UUID に変換
	mesUUID, err := uuid.Parse(mesID)
	if err != nil {
		return nil, err
	}

	// 指定された ID のレコードを取得
	mes, err := repo.GetSchMesByID(mesUUID)
	if err != nil {
		return nil, err
	}

	return mes, nil
}

// 新たな予約投稿メッセージを生成し、DB に登録
func ResisterSchMes(repo repository.Repository, id uuid.UUID, userID string, userUUIDstring string, messageID string, time time.Time, channelID string, body string) (*model.SchMes, error) {
	// チャンネル ID を UUID に変換
	channelUUID, err := uuid.Parse(channelID)
	if err != nil {
		return nil, err
	}
	userUUID, err := uuid.Parse(userUUIDstring)
	if err != nil {
		return nil, err
	}
	messageUUID, err := uuid.Parse(messageID)
	if err != nil {
		return nil, err
	}

	// 新たな SchMes 構造体型変数を生成
	schMes, err := generateSchMes(id, userID, time, userUUID, messageUUID, channelUUID, body)
	if err != nil {
		return nil, err
	}

	// DB に登録
	err = repo.ResisterSchMes(schMes)
	if err != nil {
		return nil, err
	}

	return schMes, nil
}

// 新たな SchMes 構造体型変数を生成
func generateSchMes(id uuid.UUID, userID string, time time.Time, userUUID uuid.UUID, messageUUID uuid.UUID, channelID uuid.UUID, body string) (*model.SchMes, error) {

	// SchMes 構造体型変数を生成
	return &model.SchMes{
		ID:        id,
		UserID:    userID,
		Time:      time,
		MessageID: messageUUID,
		UserUUID:  userUUID,
		ChannelID: channelID,
		Body:      body,
	}, nil
}

// 指定された ID の予約投稿メッセージを DB から削除
func DeleteSchMesByID(repo repository.Repository, api *api.API, mesID string, userID string) error {
	// ID を UUID に変換
	mesUUID, err := uuid.Parse(mesID)
	if err != nil {
		return err
	}

	// 指定された ID のレコードを検索 (存在しない ID の検証)
	mes, err := repo.GetSchMesByID(mesUUID)
	if err != nil {
		return err
	}

	// 予約したユーザーと削除を試みたユーザーが一致するか検証
	if mes.UserID != userID {
		return ErrUserNotMatch
	}

	// DB から削除
	err = repo.DeleteSchMesByID(mesUUID)
	if err != nil {
		return err
	}

	return nil
}

// 指定された メッセージID の予約投稿メッセージを DB から削除
func DeleteSchMesByMessageID(repo repository.Repository, api *api.API, mesID string, userID string) error {
	// ID を UUID に変換
	mesUUID, err := uuid.Parse(mesID)
	if err != nil {
		return err
	}

	// 指定された ID のレコードを検索
	mes, err := repo.GetSchMesByMessageID(mesUUID)
	if err != nil {
		return err
	}

	// 予約したユーザーと削除を試みたユーザーが一致するか検証
	if mes.UserUUID.String() != userID {
		return ErrUserNotMatch
	}

	// DB から削除
	err = repo.DeleteSchMesByID(mes.ID)
	if err != nil {
		return err
	}

	return nil
}

// 予約投稿メッセージを更新
func UpdateSchMes(repo repository.Repository, ID string, time *time.Time, channelID *string, body *string) (*model.SchMes, error) {
	// 指定された ID のレコードを取得
	schMes, err := GetSchMesByID(repo, ID)
	if err != nil {
		return nil, err
	}

	// 各フィールドを更新する
	if time != nil {
		schMes.Time = *time
	}
	if channelID != nil {
		// チャンネル ID を UUID に変換
		channelUUID, err := uuid.Parse(*channelID)
		if err != nil {
			return nil, err
		}
		schMes.ChannelID = channelUUID
	}
	if body != nil {
		schMes.Body = *body
	}

	// DB を更新
	err = repo.UpdateSchMes(schMes)
	if err != nil {
		return nil, err
	}

	return schMes, nil
}
