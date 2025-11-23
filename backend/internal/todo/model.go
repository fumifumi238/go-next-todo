package todo

import "time"

// Todo は ToDoタスクのデータベース構造体を表します。
// JSONタグ: クライアントとの通信用
// bindingタグ: Ginでのリクエストバリデーション用 (例: titleは必須)
type Todo struct {
	// ID: 主キー (自動採番されるため、JSONではomitemptyを付けることが多い)
	ID int `json:"id,omitempty"`

	// Title: タスクのタイトル（必須項目）
	Title string `json:"title" binding:"required"`

	// Completed: 完了状態 (0/false, 1/true)
	Completed bool `json:"completed"`

	// CreatedAt: 作成日時
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt: 更新日時 (追加することが多いが、ここでは一旦省略)
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// 💡 補足:
// - `binding:"required"` は、Ginのc.ShouldBindJSON()が呼ばれたときに、
//   JSONボディにこのフィールドが存在しない、またはゼロ値(空文字列)だった場合にエラーを発生させます。
// - `json:"id,omitempty"` は、IDがゼロ値(0)の場合、JSONからこのフィールドを除外します。
