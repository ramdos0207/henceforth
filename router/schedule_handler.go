package router

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
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

// 時刻・場所指定のコマンドハンドラー
func timeplaceHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// メッセージをパースし、要素を取得
	originalTime := strings.SplitN(req.GetText(), "\n", 4)[1]
	body := strings.SplitN(req.GetText(), "\n", 4)[3]

	distString := strings.SplitN(req.GetText(), "\n", 4)[2]
	re1 := regexp.MustCompile(`"raw":"([^"]*)"`).FindStringSubmatch(distString)
	re2 := regexp.MustCompile(`"id":"([^"]*)"`).FindStringSubmatch(distString)
	fmt.Println(re1)
	fmt.Println(re2)
	if len(re1) == 0 || len(re2) == 0 {
		service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("チャンネル情報が不正です"))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: "チャンネル情報が不正です"})
	}
	distChannel := re1[1]
	distChannelID := re2[1]
	formattedTime, err := service.Askllm(createTimeConvertPrompt(originalTime))
	if err != nil {
		service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("LLMによる時刻のパースに失敗しました\n%s", err))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}
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

// リピート・場所指定のコマンドハンドラー
func repeatplaceHandler(c echo.Context, api *api.API, repo repository.Repository, req *event.MessageEvent) error {
	// メッセージをパースし、要素を取得
	originalTime := strings.SplitN(req.GetText(), "\n", 4)[1]
	body := strings.SplitN(req.GetText(), "\n", 4)[3]
	distString := strings.SplitN(req.GetText(), "\n", 4)[2]
	re1 := regexp.MustCompile(`"raw":"([^"]*)"`).FindStringSubmatch(distString)
	re2 := regexp.MustCompile(`"id":"([^"]*)"`).FindStringSubmatch(distString)
	fmt.Println(re1)
	fmt.Println(re2)
	if len(re1) == 0 || len(re2) == 0 {
		service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("チャンネル情報が不正です"))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: "チャンネル情報が不正です"})
	}
	distChannel := re1[1]
	distChannelID := re2[1]
	formattedTime, err := service.Askllm(createRepeatConvertPrompt(originalTime))
	if err != nil {
		service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("LLMによる時刻のパースに失敗しました\n%s", err))
		return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
	}
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

			id, err := uuid.NewRandom()
			if err != nil {
				service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("UUIDの生成に失敗しました\n%s", err))
			}
			// 確認メッセージを生成
			confirmMes = service.CreateSchMesPeriodicCreatedEditedMessage(*parsedTime, distChannel, *body, id, repeat)

			// 確認メッセージを送信
			messageUUID, err := api.SendMessage(req.GetChannelID(), confirmMes)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
			}
			api.AddStamp(messageUUID, os.Getenv("DELETE_STAMP_UUID"))
			fmt.Println(messageUUID)
			userUUID := req.Message.User.ID
			// 定期投稿メッセージをDB に 登録
			_, err = service.ResisterSchMesPeriodic(repo, id, req.GetUserID(), userUUID, messageUUID, *parsedTime, *distChannelID, *body, repeat)
			if err != nil {
				service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s", err))
				return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
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
		// ID を生成
		id, err := uuid.NewRandom()
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("UUIDの生成に失敗しました\n%s", err))
			return c.JSON(http.StatusBadRequest, errorMessage{Message: err.Error()})
		}
		// 確認メッセージを生成
		confirmMes = service.CreateSchMesCreatedEditedMessage(*parsedTime, distChannel, *body, id)

		// 確認メッセージを送信
		messageUUID, err := api.SendMessage(req.GetChannelID(), confirmMes)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errorMessage{Message: fmt.Sprintf("failed to send message: %s", err)})
		}
		fmt.Println(messageUUID)
		api.AddStamp(messageUUID, os.Getenv("DELETE_STAMP_UUID"))
		userUUID := req.Message.User.ID
		// 予約投稿メッセージを DB に登録
		_, err = service.ResisterSchMes(repo, id, req.GetUserID(), userUUID, messageUUID, *parsedTime, *distChannelID, *body)
		if err != nil {
			service.SendCreateErrorMessage(api, req.GetChannelID(), fmt.Errorf("DB エラーです\n%s", err))
			return c.JSON(http.StatusInternalServerError, errorMessage{Message: err.Error()})
		}

	}

	return c.NoContent(http.StatusNoContent)
}

// 時刻パース用のプロンプトを作成
func createTimeConvertPrompt(originalTime string) string {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	nextweek := now.AddDate(0, 0, 7)
	return fmt.Sprintf(`ユーザーから与えられた予約投稿の時刻を、指定されたJSONスキーマに従って返してください。
現在時刻は%sです。year・month・day・hour・minuteには具体的な整数をそれぞれ入れ、weekdaysはnullにしてください。
例:
「明日の朝」→{"year":%d,"month":%d,"day":%d,"hour":8,"minute":0,"weekdays":null}
「夕方」→{"year":%d,"month":%d,"day":%d,"hour":17,"minute":0,"weekdays":null}
「来週」→{"year":%d,"month":%d,"day":%d,"hour":8,"minute":0,"weekdays":null}
ユーザーから与えられた時刻: %s`, now.Format("2006/01/02/15:04"),
		tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), now.Year(), now.Month(), now.Day(),
		nextweek.Year(), nextweek.Month(), nextweek.Day(), originalTime)
}

// リピート指定パース用のプロンプトを作成
func createRepeatConvertPrompt(originalTime string) string {
	now := time.Now()
	return fmt.Sprintf(`ユーザーから与えられた定期投稿の時刻を、指定されたJSONスキーマに従って返してください。
yearは常にnullにしてください。month・day・hour・minuteはそれぞれ整数で指定し、いつでも良い項目はnullにしてください。
weekdaysは日曜日が0、月曜日が1、火曜日が2、水曜日が3、木曜日が4、金曜日が5、土曜日が6の整数配列です。重複を含めず、曜日指定なし・毎日の場合はnullにしてください。
現在時刻は%sです。ユーザーが日付や曜日のみを指定した場合は現在の時・分を指定してください。
例:
「毎年12月31日の夕方」→{"year":null,"month":12,"day":31,"hour":17,"minute":0,"weekdays":null}
「平日」→{"year":null,"month":null,"day":null,"hour":%d,"minute":%d,"weekdays":[1,2,3,4,5]}
「毎週水曜の夜」→{"year":null,"month":null,"day":null,"hour":20,"minute":0,"weekdays":[3]}
「13日」→{"year":null,"month":null,"day":13,"hour":%d,"minute":%d,"weekdays":null}
「毎朝」→{"year":null,"month":null,"day":null,"hour":8,"minute":0,"weekdays":null}
ユーザーから与えられた予約情報: %s`, now.Format("2006/01/02/15:04"), now.Hour(), now.Minute(), now.Hour(), now.Minute(), originalTime)
}
