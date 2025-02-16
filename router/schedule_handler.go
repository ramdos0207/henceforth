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

// 時刻指定のみのコマンドハンドラー
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

// リピート指定のみのコマンドハンドラー
func repeatonlyHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// メッセージをパースし、要素を取得
	originalTime := strings.SplitN(req.GetText(), "\n", 3)[1]
	body := strings.SplitN(req.GetText(), "\n", 3)[2]
	distChannel := "このチャンネル"
	formattedTime, err := service.Askllm(createRepeatConvertPrompt(originalTime))
	if err != nil {
		service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("LLMによる時刻のパースに失敗しました\n%s", err))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}
	distChannelID := req.GetChannelID()
	return commonScheduleProcess(&formattedTime, &distChannel, &distChannelID, &body, nil, c, api, repo, req)
}

// 各コマンドハンドラーに共通する処理
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
			messageUUID := "023b8b87-3229-43df-9129-11805b87f307"
			userUUID := req.Message.User.ID
			// 定期投稿メッセージをDB に 登録
			schMesPeriodic, err := service.ResisterSchMesPeriodic(repo, req.GetUserID(), messageUUID, userUUID, *parsedTime, *distChannelID, *body, repeat)
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
		messageUUID := "023b8b87-3229-43df-9129-11805b87f307"
		userUUID := req.Message.User.ID

		// 予約投稿メッセージを DB に登録
		schMes, err := service.ResisterSchMes(repo, req.GetUserID(), userUUID, messageUUID, *parsedTime, *distChannelID, *body)
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

// 時刻パース用のプロンプトを作成
func createTimeConvertPrompt(originalTime string) string {
	tomorrow := time.Now().AddDate(0, 0, 1)
	answer1 := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 8, 0, 0, 0, time.Local).Format("2006/01/02/15:04")
	answer2 := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 17, 0, 0, 0, time.Local).Format("2006/01/02/15:04")
	nextweek := time.Now().AddDate(0, 0, 7)
	anser3 := time.Date(nextweek.Year(), nextweek.Month(), nextweek.Day(), 8, 0, 0, 0, time.Local).Format("2006/01/02/15:04")
	return fmt.Sprintf("あなたの仕事は、ユーザーから与えられた時刻をフォーマットすることです。現在時刻は%sです。時刻以外は何も返さないでください。いくつか例を示します。「明日の朝」→「%s」、「夕方」→「%s」、「来週」→「%s」。ユーザーから与えられた時刻: %s", time.Now().Format("2006/01/02/15:04"), answer1, answer2, anser3, originalTime)
}

// リピート指定パース用のプロンプトを作成
func createRepeatConvertPrompt(originalTime string) string {
	currenttime := time.Now().Format("15:04")
	return fmt.Sprintf("あなたの仕事は、ユーザーから与えられた反復タスクの予約を「年/月/日/時:分/曜日」形式にフォーマットすることです。曜日は日曜日が0、月曜日が1、火曜日が2、水曜日が3、木曜日が4、金曜日が5、土曜日が6で、&を用いて複数指定できます。年・月・日・時・分・曜日に関して、いつでも良い場合は「*」と指定してください。現在時刻は%sなので、ユーザーが日付のみを指定した場合は現在時刻を指定してください。フォーマットされた結果以外は何も返さないでください。いくつか例を示します。「毎年12月31日の夕方」→「*/12/31/17:00/*」、「平日」→「*/*/*/%s/1&2&3&4&5」、「毎週水曜の夜」→「*/*/*/20:00/3」、「13日」→「*/*/13/%s/3」。ユーザーから与えられた予約情報: %s", time.Now().Format("2006/01/02/15:04"), currenttime, currenttime, originalTime)
}
