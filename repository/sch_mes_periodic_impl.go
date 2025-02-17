package repository

import (
	"github.com/google/uuid"
	"github.com/logica0419/scheduled-messenger-bot/model"
)

// 定期投稿メッセージを全取得
func (repo *GormRepository) GetSchMesPeriodicAll() ([]*model.SchMesPeriodic, error) {
	// キャッシュが存在すればそこから読み取って返す
	content := repo.getSchMesPeriodicCache()
	if content != nil {
		return content, nil
	}

	// 空のメッセージ構造体の変数を作成
	var schMesPeriodic []*model.SchMesPeriodic

	// レコードを取得
	res := repo.getTx().Find(&schMesPeriodic)
	if res.Error != nil {
		return nil, res.Error
	}

	// キャッシュに格納
	repo.setSchMesPeriodicCache(schMesPeriodic)

	return schMesPeriodic, nil
}

// 指定された ID の定期投稿メッセージのレコードを取得
func (repo *GormRepository) GetSchMesPeriodicByID(mesID uuid.UUID) (*model.SchMesPeriodic, error) {
	// キャッシュが存在すればそこから読み取って返す
	content := repo.getSchMesPeriodicCache()
	for _, v := range content {
		if v.ID == mesID {
			return v, nil
		}
	}

	// 空のメッセージ構造体の変数を作成
	var schMesPeriodic *model.SchMesPeriodic

	// レコードを取得
	res := repo.getTx().Where("id = ?", mesID).Take(&schMesPeriodic)
	if res.Error != nil {
		return nil, res.Error
	}

	return schMesPeriodic, nil
}

// 指定された ID の定期投稿メッセージのレコードを取得
func (repo *GormRepository) GetSchMesPeriodicByMessageID(mesID uuid.UUID) (*model.SchMesPeriodic, error) {
	// キャッシュが存在すればそこから読み取って返す
	content := repo.getSchMesPeriodicCache()
	for _, v := range content {
		if v.MessageID == mesID {
			return v, nil
		}
	}

	// 空のメッセージ構造体の変数を作成
	var schMesPeriodic *model.SchMesPeriodic

	// レコードを取得
	res := repo.getTx().Where("message_id = ?", mesID).Take(&schMesPeriodic)
	if res.Error != nil {
		return nil, res.Error
	}

	return schMesPeriodic, nil
}

// 指定された UserID の予約投稿メッセージのレコードを全取得
func (repo *GormRepository) GetSchMesPeriodicByUserID(userID string) ([]*model.SchMesPeriodic, error) {
	// 空のメッセージ構造体の変数を作成
	var schMesPeriodic []*model.SchMesPeriodic

	// キャッシュが存在すればそこから読み取って返す
	content := repo.getSchMesPeriodicCache()
	if content != nil {
		for _, v := range content {
			if v.UserID == userID {
				schMesPeriodic = append(schMesPeriodic, v)
			}
		}

		return schMesPeriodic, nil
	}

	// レコードを取得
	res := repo.getTx().Where("user_id = ?", userID).Find(&schMesPeriodic)
	if res.Error != nil {
		return nil, res.Error
	}

	return schMesPeriodic, nil
}

// 定期投稿メッセージのレコードを新規作成
func (repo *GormRepository) ResisterSchMesPeriodic(schMesPeriodic *model.SchMesPeriodic) error {
	// レコードを作成
	res := repo.getTx().Create(schMesPeriodic)
	if res.Error != nil {
		return res.Error
	}

	// キャッシュから全レコードを削除
	repo.deleteSchMesPeriodicCache()

	return nil
}

// 指定された ID の定期投稿メッセージのレコードを削除
func (repo *GormRepository) DeleteSchMesPeriodicByID(mesID uuid.UUID) error {
	// ID のみの定期投稿メッセージ構造体の変数を作成 (primary key 指定のため)
	schMesPeriodic := model.SchMesPeriodic{
		ID: mesID,
	}

	// レコードを削除
	res := repo.getTx().Delete(&schMesPeriodic)
	if res.Error != nil {
		return res.Error
	}

	// キャッシュから全レコードを削除
	repo.deleteSchMesPeriodicCache()

	return nil
}

// 定期投稿メッセージのレコードを更新
func (repo *GormRepository) UpdateSchMesPeriodic(schMesPeriodic *model.SchMesPeriodic) error {
	// レコードを更新
	res := repo.getTx().Model(schMesPeriodic).Select("*").Updates(schMesPeriodic)
	if res.Error != nil {
		return res.Error
	}

	// キャッシュから全レコードを削除
	repo.deleteSchMesPeriodicCache()

	return nil
}
