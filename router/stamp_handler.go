package router

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/logica0419/scheduled-messenger-bot/model/event"
	"github.com/logica0419/scheduled-messenger-bot/repository"
	"github.com/logica0419/scheduled-messenger-bot/service"
	"github.com/logica0419/scheduled-messenger-bot/service/api"
	"gorm.io/gorm"
)

func stampEventHandler(c echo.Context, api *api.API, repo repository.Repository) error {
	// リクエストボディの取得
	req := &event.StampEvent{}
	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to get request body: %s", err)})
	}

	fmt.Println(req)
	stampUUID := os.Getenv("DELETE_STAMP_UUID")
	channnelID := "dummy"
	// 各スタンプについて検証
	for _, stamp := range req.Stamps {
		if stamp.StampID == stampUUID {
			// 予約投稿メッセージを DB から削除
			err = service.DeleteSchMesByMessageID(repo, api, req.MessageID, stamp.UserID)
			// 指定した ID のメッセージが存在しない場合定期投稿の削除を試みる
			if errors.Is(err, gorm.ErrRecordNotFound) {
				goto periodic
			} else if err != nil {
				// 指定した ID が無効な場合エラーメッセージを送信
				if uuid.IsInvalidLengthError(err) {
					service.SendDeleteErrorMessage(api, channnelID, fmt.Errorf("存在しないIDです\n"))
					return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
				}
				// 登録したユーザーと削除を試みたユーザーが違う場合エラーメッセージを送信
				if errors.Is(err, service.ErrUserNotMatch) {
					service.SendDeleteErrorMessage(api, channnelID, fmt.Errorf("予約メッセージは予約したユーザーしか削除できません\n"))
					return c.JSON(http.StatusForbidden, errorMessage{Message: err.Error()})
				}
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
			}

			goto message

		periodic: // 定期投稿メッセージを DB から削除
			err = service.DeleteSchMesPeriodicByMessageID(repo, api, req.MessageID, stamp.UserID)
			if err != nil {
				// 指定した ID のメッセージが存在しない場合エラーメッセージを送信
				if errors.Is(err, gorm.ErrRecordNotFound) {
					service.SendDeleteErrorMessage(api, channnelID, fmt.Errorf("存在しないIDです\n"))
					return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
				}
				// 予約したユーザーと削除を試みたユーザーが違う場合エラーメッセージを送信
				if errors.Is(err, service.ErrUserNotMatch) {
					service.SendDeleteErrorMessage(api, channnelID, fmt.Errorf("予約メッセージは予約したユーザーしか削除できません\n"))
					return c.JSON(http.StatusForbidden, errorMessage{Message: err.Error()})
				}
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
			}

			goto message

		message: // 確認メッセージを送信
			mes := "スタンプにより予約メッセージがキャンセルされました"
			_, err = api.SendMessage(channnelID, mes)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
			}

			return c.NoContent(http.StatusNoContent)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
