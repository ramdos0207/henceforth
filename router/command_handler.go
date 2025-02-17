package router

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/logica0419/scheduled-messenger-bot/model"
	"github.com/logica0419/scheduled-messenger-bot/model/event"
	"github.com/logica0419/scheduled-messenger-bot/repository"
	"github.com/logica0419/scheduled-messenger-bot/service"
	"github.com/logica0419/scheduled-messenger-bot/service/api"
	"github.com/logica0419/scheduled-messenger-bot/service/parser"
	"gorm.io/gorm"
)

// help コマンドハンドラー
func helpHandler(c echo.Context, api *api.API, req *event.MessageEvent) error {
	mes := service.CreateHelpMessage()

	_, err := api.SendMessage(req.GetChannelID(), mes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// edit コマンドハンドラー
func editHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// メッセージをパースし、要素を取得
	id, postTime, distChannel, distChannelID, body, repeat, err := parser.ParseEditCommand(req)
	if err != nil {
		service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("%s", err))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	// 確認メッセージ
	var confirmMes string

	// 予約投稿かどうかを確認
	isPeriodic := false
	// 予約投稿を ID で取得
	_, err = service.GetSchMesByID(repo, *id)
	// 指定した ID のメッセージが存在しない場合定期投稿の取得を試みる
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_, err = service.GetSchMesPeriodicByID(repo, *id)
		if err != nil {
			// 指定した ID のメッセージが存在しない場合エラーメッセージを送信
			if errors.Is(err, gorm.ErrRecordNotFound) {
				service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("存在しないIDです\n"))
			}
			// 登録したユーザーと編集を試みたユーザーが違う場合エラーメッセージを送信
			if errors.Is(err, service.ErrUserNotMatch) {
				service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("予約メッセージは予約したユーザーしか編集できません\n"))
				return c.JSON(http.StatusForbidden, errorMessage{Message: err.Error()})
			}
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}

		isPeriodic = true
	} else if err != nil {
		// 指定した ID が無効な場合エラーメッセージを送信
		if uuid.IsInvalidLengthError(err) {
			service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("存在しないIDです\n"))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}
		// 登録したユーザーと編集を試みたユーザーが違う場合エラーメッセージを送信
		if errors.Is(err, service.ErrUserNotMatch) {
			service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("予約メッセージは予約したユーザーしか編集できません\n"))
			return c.JSON(http.StatusForbidden, errorMessage{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
	}

	// 投稿の種類で処理を分岐
	if isPeriodic { // 定期投稿
		var parsedTime *model.PeriodicTime
		// 時間が nil でなければパース
		if postTime != nil {
			parsedTimes, err := parser.TimeParsePeriodic(postTime)
			if err != nil {
				service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("無効な時間表記です\n%s\n", err))
				return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
			}
			// 曜日は複数指定されていたらエラーメッセージを送信
			if len(parsedTimes) > 1 {
				service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("無効な時間表記です\n編集で複数の曜日を指定することはできません\n"))
				return c.JSON(http.StatusBadRequest, errorMessage{Message: "編集で複数の曜日を指定することはできません"})
			}

			parsedTime = parsedTimes[0]
		}

		// 定期投稿メッセージを更新
		schMesPeriodic, err := service.UpdateSchMesPeriodic(repo, *id, parsedTime, distChannelID, body, repeat)
		if err != nil {
			service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s\n", err))
			return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
		}

		// 確認メッセージを生成
		confirmMes = service.CreateSchMesPeriodicCreatedEditedMessage(schMesPeriodic.Time, distChannel, schMesPeriodic.Body, schMesPeriodic.ID, schMesPeriodic.Repeat)

	} else { // 予約投稿
		// repeat が入力されていたらエラーメッセージを送る
		if repeat != nil {
			service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("予約投稿でリピートは使用できません\n"))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: "予約投稿でリピートは使用できません\n"})
		}

		var parsedTime *time.Time
		// 時間が nil でなければパース
		if postTime != nil {
			parsedTime, err = parser.TimeParse(postTime)
			if err != nil {
				service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("無効な時間表記です\n%s\n", err))
				return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
			}
		}

		// 予約投稿メッセージを更新
		schMes, err := service.UpdateSchMes(repo, *id, parsedTime, distChannelID, body)
		if err != nil {
			service.SendEditErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s\n", err))
			return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
		}

		// 確認メッセージを生成
		confirmMes = service.CreateSchMesCreatedEditedMessage(schMes.Time, distChannel, schMes.Body, schMes.ID)
	}

	// 確認メッセージを送信
	_, err = api.SendMessage(req.GetChannelID(), confirmMes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// delete コマンドハンドラー
func deleteHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// メッセージをパースし、要素を取得
	id, err := parser.ParseDeleteCommand(req)
	if err != nil {
		service.SendDeleteErrorMessage(api, req.GetChannelID(), fmt.Errorf("メッセージをパースできません\n%s", err))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}

	// 予約投稿メッセージを DB から削除
	err = service.DeleteSchMesByID(repo, api, *id, req.GetUserID())
	// 指定した ID のメッセージが存在しない場合定期投稿の削除を試みる
	if errors.Is(err, gorm.ErrRecordNotFound) {
		goto periodic
	} else if err != nil {
		// 指定した ID が無効な場合エラーメッセージを送信
		if uuid.IsInvalidLengthError(err) {
			service.SendDeleteErrorMessage(api, req.GetChannelID(), fmt.Errorf("存在しないIDです\n"))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}
		// 登録したユーザーと削除を試みたユーザーが違う場合エラーメッセージを送信
		if errors.Is(err, service.ErrUserNotMatch) {
			service.SendDeleteErrorMessage(api, req.GetChannelID(), fmt.Errorf("予約メッセージは予約したユーザーしか削除できません\n"))
			return c.JSON(http.StatusForbidden, errorMessage{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
	}

	goto message

periodic: // 定期投稿メッセージを DB から削除
	err = service.DeleteSchMesPeriodicByID(repo, api, *id, req.GetUserID())
	if err != nil {
		// 指定した ID のメッセージが存在しない場合エラーメッセージを送信
		if errors.Is(err, gorm.ErrRecordNotFound) {
			service.SendDeleteErrorMessage(api, req.GetChannelID(), fmt.Errorf("存在しないIDです\n"))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}
		// 予約したユーザーと削除を試みたユーザーが違う場合エラーメッセージを送信
		if errors.Is(err, service.ErrUserNotMatch) {
			service.SendDeleteErrorMessage(api, req.GetChannelID(), fmt.Errorf("予約メッセージは予約したユーザーしか削除できません\n"))
			return c.JSON(http.StatusForbidden, errorMessage{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
	}

	goto message

message: // 確認メッセージを送信
	mes := service.CreateSchMesDeletedMessage(*id)
	_, err = api.SendMessage(req.GetChannelID(), mes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// list コマンドハンドラー
func listHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// ユーザー ID を取得
	userID := req.GetUserID()

	// 予約投稿メッセージを DB から取得
	mesList, err := repo.GetSchMesByUserID(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
	}

	// 定期投稿メッセージを DB から取得
	mesListPeriodic, err := repo.GetSchMesPeriodicByUserID(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
	}

	// 予約メッセージリストを送信
	mes := service.CreateScheduleListMessage(mesList, mesListPeriodic)
	_, err = api.SendMessage(req.GetChannelID(), mes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// join コマンドハンドラー
func joinHandler(c echo.Context, api *api.API, req *event.MessageEvent) error {
	// チャンネルに JOIN する
	err := api.JoinChannel(req.GetChannelID())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to join the channel: %s", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// leave コマンドハンドラー
func leaveHandler(c echo.Context, api *api.API, req *event.MessageEvent) error {
	// チャンネルから LEAVE する
	err := api.LeaveChannel(req.GetChannelID())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to leave the channel: %s", err)})
	}

	return c.NoContent(http.StatusNoContent)
}
