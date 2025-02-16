package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/logica0419/scheduled-messenger-bot/config"
)

// 共通 API ベース URL
const baseUrl = "https://q.trap.jp/api/v3"

// API クライアント構造体
type API struct {
	client *http.Client
	config *config.Config
}

// API クライアントの取得
func GetApi(c *config.Config) *API {
	client := new(http.Client)
	api := &API{client: client, config: c}

	return api
}

// リクエストのヘッダにトークンを付与
func setTokenHeader(req *http.Request, api *API) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.config.Bot_Access_Token))
}

// リクエストのヘッダに JSON の Content-Type を付与
func setJsonHeader(req *http.Request) {
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
}

// POST リクエストを送信
func (api *API) post(url string, body interface{}) ([]byte, error) {
	// ボディをバイト列に変換
	byteBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// 変換したボディを載せて POST リクエストを作成
	req, err := http.NewRequest("POST", url, bytes.NewReader(byteBody))
	if err != nil {
		return nil, err
	}

	// ヘッダーを設定
	setTokenHeader(req, api)
	setJsonHeader(req)

	// リクエストを送信
	res, err := api.client.Do(req)
	// Dev Mode の場合レスポンスをログに出す
	if api.config.Dev_Mode {
		log.Println(*res)
	}
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf(res.Status)
	}
	defer res.Body.Close()
	resbody, _ := io.ReadAll(res.Body)
	return resbody, nil
}
