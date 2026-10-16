package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/logica0419/scheduled-messenger-bot/service/parser"
)

const llmEndpoint = "https://llm-proxy.trap.jp/chat/completions"

const llmResponseFormat = `{
	"type": "json_schema",
	"json_schema": {
		"name": "schedule_time",
		"strict": true,
		"schema": {
			"type": "object",
			"properties": {
				"year": {"type": ["integer", "null"], "minimum": 1, "maximum": 9999, "description": "予約投稿の年。定期投稿ではnull。"},
				"month": {"type": ["integer", "null"], "minimum": 1, "maximum": 12, "description": "月。定期投稿で指定がない場合はnull。"},
				"day": {"type": ["integer", "null"], "minimum": 1, "maximum": 31, "description": "日。定期投稿で指定がない場合はnull。"},
				"hour": {"type": ["integer", "null"], "minimum": 0, "maximum": 23, "description": "時。定期投稿で指定がない場合はnull。"},
				"minute": {"type": ["integer", "null"], "minimum": 0, "maximum": 59, "description": "分。定期投稿で指定がない場合はnull。"},
				"weekdays": {
					"type": ["array", "null"],
					"items": {"type": "integer", "minimum": 0, "maximum": 6},
					"minItems": 1,
					"maxItems": 6,
					"description": "定期投稿する曜日の重複しない配列。日曜が0、土曜が6。予約投稿または曜日指定なしの場合はnull。"
				}
			},
			"required": ["year", "month", "day", "hour", "minute", "weekdays"],
			"additionalProperties": false
		}
	}
}`

var llmHTTPClient = &http.Client{Timeout: 30 * time.Second}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmScheduleTime struct {
	Year     *int   `json:"year"`
	Month    *int   `json:"month"`
	Day      *int   `json:"day"`
	Hour     *int   `json:"hour"`
	Minute   *int   `json:"minute"`
	Weekdays []*int `json:"weekdays"`
}

// 生成AIに質問を投げ、構造化出力からフォーマット済みの時刻を取得する
func Askllm(question string) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv("LLM_PROXY_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("LLM_PROXY_API_KEYが設定されていません")
	}

	payload := struct {
		Model          string          `json:"model"`
		Messages       []llmMessage    `json:"messages"`
		ResponseFormat json.RawMessage `json:"response_format"`
	}{
		Model:          "recommended-tiny",
		Messages:       []llmMessage{{Role: "user", Content: question}},
		ResponseFormat: json.RawMessage(llmResponseFormat),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("生成AIリクエストの作成に失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, llmEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("生成AIリクエストの作成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	fmt.Println(question)
	resp, err := llmHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("生成AI APIの呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("生成AI APIの呼び出しに失敗しました: HTTP %d", resp.StatusCode)
	}

	var completion struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return "", fmt.Errorf("生成AI APIの応答の解析に失敗しました: %w", err)
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("生成AI APIの応答に回答が含まれていません")
	}
	choice := completion.Choices[0]
	if choice.Message.Refusal != "" {
		return "", fmt.Errorf("生成AI APIが回答を拒否しました")
	}
	if choice.FinishReason != "stop" {
		return "", fmt.Errorf("生成AI APIの応答が完了していません: finish_reason=%q", choice.FinishReason)
	}
	if strings.TrimSpace(choice.Message.Content) == "" {
		return "", fmt.Errorf("生成AI APIの応答に回答が含まれていません")
	}

	result, err := decodeLLMScheduleTime(choice.Message.Content)
	if err != nil {
		return "", err
	}
	fmt.Println(result)
	return result, nil
}

func decodeLLMScheduleTime(content string) (string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &fields); err != nil {
		return "", fmt.Errorf("生成AI APIの構造化出力の解析に失敗しました: %w", err)
	}
	required := []string{"year", "month", "day", "hour", "minute", "weekdays"}
	for _, name := range required {
		if _, ok := fields[name]; !ok {
			return "", fmt.Errorf("生成AI APIの構造化出力に%sフィールドが含まれていません", name)
		}
	}
	if len(fields) != len(required) {
		return "", fmt.Errorf("生成AI APIの構造化出力に余分なフィールドが含まれています")
	}
	var result llmScheduleTime
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return "", fmt.Errorf("生成AI APIの構造化出力の解析に失敗しました: %w", err)
	}
	return result.format()
}

// nullをワイルドカードに変換し、既存パーサーで扱う時刻表記を組み立てる。
func (value llmScheduleTime) format() (string, error) {
	formatted := fmt.Sprintf("%s/%s/%s/%s:%s",
		llmTimePart(value.Year, 4), llmTimePart(value.Month, 2), llmTimePart(value.Day, 2),
		llmTimePart(value.Hour, 2), llmTimePart(value.Minute, 2))
	if value.Year != nil {
		if *value.Year < 1 || *value.Year > 9999 || value.Month == nil || value.Day == nil ||
			value.Hour == nil || value.Minute == nil || value.Weekdays != nil {
			return "", fmt.Errorf("生成AI APIの構造化出力の予約時刻が不正です")
		}
		if _, err := time.Parse("2006/01/02/15:04", formatted); err != nil {
			return "", fmt.Errorf("生成AI APIの構造化出力の予約時刻が不正です: %w", err)
		}
		return formatted, nil
	}
	weekdayPart := "*"
	if value.Weekdays != nil {
		if len(value.Weekdays) == 0 {
			return "", fmt.Errorf("生成AI APIの構造化出力の曜日配列が空です")
		}
		weekdays := make([]string, len(value.Weekdays))
		for i, day := range value.Weekdays {
			if day == nil {
				return "", fmt.Errorf("生成AI APIの構造化出力の曜日にnullは使えません")
			}
			weekdays[i] = strconv.Itoa(*day)
		}
		weekdayPart = strings.Join(weekdays, "&")
	}
	formatted += "/" + weekdayPart
	if _, err := parser.TimeParsePeriodic(&formatted); err != nil {
		return "", fmt.Errorf("生成AI APIの構造化出力の定期投稿時刻が不正です: %w", err)
	}
	return formatted, nil
}

func llmTimePart(value *int, width int) string {
	if value == nil {
		return "*"
	}
	return fmt.Sprintf("%0*d", width, *value)
}
