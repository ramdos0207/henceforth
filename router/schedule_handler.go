package router

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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
func timeonlyHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// メッセージをパースし、要素を取得
	originalTime := strings.SplitN(req.GetText(), "\n", 3)[1]
	body := strings.SplitN(req.GetText(), "\n", 3)[2]
	distChannel := "このチャンネル"
	formattedTime, err := service.Askllm(createTimeConvertPrompt(originalTime))
	if err != nil {
		service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("LLMによる時刻のパースに失敗しました\n%s", err))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}
	distChannelID := req.GetChannelID()
	return commonScheduleProcess(&formattedTime, &distChannel, &distChannelID, &body, nil, c, api, repo, req)
}
func commonScheduleProcess(time *string, distChannel *string, distChannelID *string, body *string, repeat *int, c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// 確認メッセージ
	var confirmMes string
	// 時間の表記にワイルドカードが含まれているかで処理を分岐
	if strings.Contains(*time, "*") { // 定期投稿
		// 時間をパース
		trimmedTime := strings.TrimSpace(*time)
		parsedTimes, err := parser.TimeParsePeriodic(&trimmedTime)
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("無効な時間表記です\n%s", err))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}

		for _, parsedTime := range parsedTimes {
			// 定期投稿メッセージをDB に 登録
			schMesPeriodic, err := service.ResisterSchMesPeriodic(repo, req.GetUserID(), *parsedTime, *distChannelID, *body, repeat)
			if err != nil {
				service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s", err))
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
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("予約投稿でリピートは使用できません"))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: "予約投稿でリピートは使用できません"})
		}

		// 時間をパース
		trimmedTime := strings.TrimSpace(*time)
		parsedTime, err := parser.TimeParse(&trimmedTime)
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("無効な時間表記です\n%s", err))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}

		// 予約投稿メッセージを DB に登録
		schMes, err := service.ResisterSchMes(repo, req.GetUserID(), *parsedTime, *distChannelID, *body)
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s", err))
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
func createTimeConvertPrompt(originalTime string) string {
	return fmt.Sprintf("次の日時情報を、「2006/01/02/15:04」形式に直してください。現在時刻は%sです。「明日」のような曖昧な時刻が渡された場合、できるだけ4の倍数時、例えば8:00、16:00、20:00などを返すようにしてください。ただし、ユーザーが具体的な時刻を指定した場合はそちらを優先してください。年月日時刻のいずれも省略してはいけません。時刻のみが指定された場合は今後その時刻を迎えるできるだけ早い日時を返してください。時刻以外は何も返さないでください。直すべき時刻情報:%s", time.Now().Format("2006/01/02/15:04"), originalTime)
}
