package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/logica0419/scheduled-messenger-bot/model/event"
	"github.com/logica0419/scheduled-messenger-bot/repository"
	"github.com/logica0419/scheduled-messenger-bot/service"
	"github.com/logica0419/scheduled-messenger-bot/service/api"
	"github.com/logica0419/scheduled-messenger-bot/service/parser"
)

// schedule コマンドハンドラー
func scheduleHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// メッセージをパースし、要素を取得
	time, distChannel, distChannelID, body, repeat, err := parser.ParseScheduleCommand(req)
	if err != nil {
		service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("%s", err))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}
	return commonScheduleProcess(time, distChannel, distChannelID, body, repeat, c, api, repo, req)
}
func commonScheduleProcess(time *string, distChannel *string, distChannelID *string, body *string, repeat *int, c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// 確認メッセージ
	var confirmMes string
	// 時間の表記にワイルドカードが含まれているかで処理を分岐
	if strings.Contains(*time, "*") { // 定期投稿
		// 時間をパース
		parsedTimes, err := parser.TimeParsePeriodic(time)
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("無効な時間表記です\n%s\n", err))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}

		for _, parsedTime := range parsedTimes {
			// 定期投稿メッセージをDB に 登録
			schMesPeriodic, err := service.ResisterSchMesPeriodic(repo, req.GetUserID(), *parsedTime, *distChannelID, *body, repeat)
			if err != nil {
				service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s\n", err))
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
			}

			// 確認メッセージを生成
			confirmMes = service.CreateSchMesPeriodicCreatedEditedMessage(schMesPeriodic.Time, distChannel, schMesPeriodic.Body, schMesPeriodic.ID, schMesPeriodic.Repeat)

			// 確認メッセージを送信
			err = api.SendMessage(req.GetChannelID(), confirmMes)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
			}
		}

	} else { // 予約投稿
		// repeat が入力されていたらエラーメッセージを送る
		if repeat != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("予約投稿でリピートは使用できません\n"))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: "予約投稿でリピートは使用できません\n"})
		}

		// 時間をパース
		parsedTime, err := parser.TimeParse(time)
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("無効な時間表記です\n%s\n", err))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}

		// 予約投稿メッセージを DB に登録
		schMes, err := service.ResisterSchMes(repo, req.GetUserID(), *parsedTime, *distChannelID, *body)
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s\n", err))
			return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
		}

		// 確認メッセージを生成
		confirmMes = service.CreateSchMesCreatedEditedMessage(schMes.Time, distChannel, schMes.Body, schMes.ID)

		// 確認メッセージを送信
		err = api.SendMessage(req.GetChannelID(), confirmMes)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
		}
	}

	return c.NoContent(http.StatusNoContent)
}
