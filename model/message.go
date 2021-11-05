package model

import (
	"time"

	"github.com/google/uuid"
)

// スケジュールドメッセージのモデル定義
type SchMes struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	UserID    string
	Time      time.Time `gorm:"index"`
	ChannelID uuid.UUID
	Body      string
}
