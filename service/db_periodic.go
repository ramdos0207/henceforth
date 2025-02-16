package service

import (
	"github.com/google/uuid"
	"github.com/logica0419/scheduled-messenger-bot/model"
	"github.com/logica0419/scheduled-messenger-bot/repository"
	"github.com/logica0419/scheduled-messenger-bot/service/api"
)

// 定期投稿メッセージを DB から取得
func GetSchMesPeriodicByID(repo repository.Repository, mesID string) (*model.SchMesPeriodic, error) {
	// ID を UUID に変換
	mesUUID, err := uuid.Parse(mesID)
	if err != nil {
		return nil, err
	}

	// 指定された ID のレコードを取得
	mes, err := repo.GetSchMesPeriodicByID(mesUUID)
	if err != nil {
		return nil, err
	}

	return mes, nil
}

// 新たな定期投稿メッセージを生成し、DB に登録
func ResisterSchMesPeriodic(repo repository.Repository, userID string, userUUIDstring string, MessageID string, time model.PeriodicTime, channelID string, body string, repeat *int) (*model.SchMesPeriodic, error) {
	// チャンネル ID を UUID に変換
	channelUUID, err := uuid.Parse(channelID)
	if err != nil {
		return nil, err
	}
	userUUID, err := uuid.Parse(userUUIDstring)
	if err != nil {
		return nil, err
	}
	messageUUID, err := uuid.Parse(MessageID)
	if err != nil {
		return nil, err
	}

	// 新たな SchMesPeriodic 構造体型変数を生成
	schMesPeriodic, err := generateSchMesPeriodic(userID, userUUID, messageUUID, time, channelUUID, body, repeat)
	if err != nil {
		return nil, err
	}

	// DB に登録
	err = repo.ResisterSchMesPeriodic(schMesPeriodic)
	if err != nil {
		return nil, err
	}

	return schMesPeriodic, nil
}

// 新たな SchMesPeriodic 構造体型変数を生成
func generateSchMesPeriodic(userID string, userUUID uuid.UUID, messageUUID uuid.UUID, time model.PeriodicTime, channelID uuid.UUID, body string, repeat *int) (*model.SchMesPeriodic, error) {
	// ID を生成
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// SchMes 構造体型変数を生成
	return &model.SchMesPeriodic{
		ID:        id,
		UserID:    userID,
		Time:      time,
		Repeat:    repeat,
		MessageID: messageUUID,
		UserUUID:  userUUID,
		ChannelID: channelID,
		Body:      body,
	}, nil
}

// 指定された ID の定期投稿メッセージを DB から削除
func DeleteSchMesPeriodicByID(repo repository.Repository, api *api.API, mesID string, userID string) error {
	// ID を UUID に変換
	mesUUID, err := uuid.Parse(mesID)
	if err != nil {
		return err
	}

	// 指定された ID のレコードを検索 (存在しない ID の検証)
	mes, err := repo.GetSchMesPeriodicByID(mesUUID)
	if err != nil {
		return err
	}

	// 予約したユーザーと削除を試みたユーザーが一致するか検証
	if mes.UserID != userID {
		return ErrUserNotMatch
	}

	// DB から削除
	err = repo.DeleteSchMesPeriodicByID(mesUUID)
	if err != nil {
		return err
	}

	return nil
}

// 定期投稿メッセージを更新
func UpdateSchMesPeriodic(repo repository.Repository, ID string, time *model.PeriodicTime, channelID *string, body *string, repeat *int) (*model.SchMesPeriodic, error) {
	// 指定された ID のレコードを取得
	schMesPeriodic, err := GetSchMesPeriodicByID(repo, ID)
	if err != nil {
		return nil, err
	}

	// 各フィールドを更新する
	if time != nil {
		schMesPeriodic.Time = *time
	}
	if repeat != nil {
		if *repeat < 0 {
			schMesPeriodic.Repeat = nil
		} else {
			schMesPeriodic.Repeat = repeat
		}
	}
	if channelID != nil {
		// チャンネル ID を UUID に変換
		channelUUID, err := uuid.Parse(*channelID)
		if err != nil {
			return nil, err
		}
		schMesPeriodic.ChannelID = channelUUID
	}
	if body != nil {
		schMesPeriodic.Body = *body
	}

	// DB を更新
	err = repo.UpdateSchMesPeriodic(schMesPeriodic)
	if err != nil {
		return nil, err
	}

	return schMesPeriodic, nil
}
