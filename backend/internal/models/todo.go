// Package modelsはTodoを定義します。
package models

import (
	"time"
)

type Todo struct {
	ID        int       `json:"id,omitempty"`             // 主キー
	UserID    int       `json:"user_id"`                  // 💡 追加: ユーザーID (必須)
	Title     string    `json:"title" binding:"required"` // タスクのタイトル（必須）
	Completed bool      `json:"completed"`                // 完了状態
	CreatedAt time.Time `json:"created_at"`               // 作成日時
	UpdatedAt time.Time `json:"updated_at,omitempty"`     // 💡 追加: 更新日時
}
