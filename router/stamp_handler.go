package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/logica0419/scheduled-messenger-bot/model/event"
	"github.com/logica0419/scheduled-messenger-bot/repository"
	"github.com/logica0419/scheduled-messenger-bot/service"
	"github.com/logica0419/scheduled-messenger-bot/service/api"
	"github.com/traPtitech/go-traq"
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

	client := traq.NewAPIClient(traq.NewConfiguration())
	auth := context.WithValue(context.Background(), traq.ContextAccessToken, api.Config.Bot_Access_Token)

	v, _, err := client.MessageApi.GetMessage(auth, req.MessageID).Execute()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to get message: %s", err)})
	}
	stampUUID := os.Getenv("DELETE_STAMP_UUID")
	channnelID := v.ChannelId

	// 各スタンプについて検証
	for _, stamp := range req.Stamps {
		if stamp.StampID == stampUUID {
			// 予約投稿メッセージを DB から削除
			err = service.DeleteSchMesByMessageID(repo, api, req.MessageID, stamp.UserID)
			// 指定した ID のメッセージが存在しない場合定期投稿の削除を試みる
			if errors.Is(err, gorm.ErrRecordNotFound) {
				goto periodic
			} else if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				} else if errors.Is(err, service.ErrUserNotMatch) {
					continue
				} else {
					return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
				}
			}

			goto message

		periodic: // 定期投稿メッセージを DB から削除
			err = service.DeleteSchMesPeriodicByMessageID(repo, api, req.MessageID, stamp.UserID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				} else if errors.Is(err, service.ErrUserNotMatch) {
					continue
				} else {
					return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
				}
			}

			goto message

		message: // 確認メッセージを送信
			mes := fmt.Sprintf("スタンプにより[この予約メッセージ](%s%s)がキャンセルされました",
				os.Getenv("MESSAGE_URL_PREFIX"), req.MessageID)
			_, err = api.SendMessage(channnelID, mes)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
			}

			return c.NoContent(http.StatusNoContent)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
