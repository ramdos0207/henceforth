package service

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func Askllm(question string) (string, error) {
	ctx := context.Background()
	fmt.Println(question)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY"))) //TODO configに切り替える
	if err != nil {
		return "", fmt.Errorf("生成AIクライアントの作成に失敗しました: %v", err)
	}
	defer client.Close()

	// 質問を送信して回答を取得する
	model := client.GenerativeModel("models/gemini-2.0-flash-lite-preview-02-05")
	prompt := genai.Text(question)
	resp, err := model.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("生成AI APIの呼び出しに失敗しました: %v", err)
	}
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			result += string(txt)
		}
	}
	fmt.Println(result)
	return result, nil
}
