package todo

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

// Repository はデータベース操作を行うための構造体です。
type Repository struct {
	DB *sql.DB
}

// NewRepository は新しいRepositoryインスタンスを作成します。
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// Create は新しいTodoタスクをデータベースに挿入します。
func (r *Repository) Create(t *Todo) (*Todo, error) {
	// 挿入クエリの準備
	// `completed` は bool (Go) から tinyint/boolean (MySQL) に変換されます。
	query := "INSERT INTO todos (title, completed) VALUES (?, ?)"

	// クエリの実行
	// Exec()は結果（LastInsertIdとRowsAffected）を返します。
	result, err := r.DB.Exec(query, t.Title, t.Completed)
	if err != nil {
		log.Printf("Failed to insert todo: %v", err)
		return nil, fmt.Errorf("could not insert todo: %w", err)
	}

	// 1. 自動採番されたIDを取得
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert ID: %w", err)
	}

	// 2. 作成されたオブジェクトにIDとタイムスタンプをセット
	t.ID = int(id)

	// データベースが自動で設定した created_at を取得しなくても、
	// テストのために時刻を仮に設定しておく（または、次のステップで SELECT して取得する）
	t.CreatedAt = time.Now()

	return t, nil
}

// FindAll はすべてのTodoタスクをデータベースから取得します。
func (r *Repository) FindAll() ([]*Todo, error) {
	query := "SELECT id, title, completed, created_at FROM todos ORDER BY created_at DESC"

	rows, err := r.DB.Query(query)
	if err != nil {
		log.Printf("Failed to query todos: %v", err)
		return nil, fmt.Errorf("could not query todos: %w", err)
	}
	defer rows.Close()

	var todos []*Todo
	for rows.Next() {
		var t Todo
		err := rows.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)
		if err != nil {
			log.Printf("Failed to scan todo: %v", err)
			return nil, fmt.Errorf("could not scan todo: %w", err)
		}
		todos = append(todos, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating todos: %w", err)
	}

	return todos, nil
}

// ErrTodoNotFound はTODOが見つからない場合のエラーです。
var ErrTodoNotFound = errors.New("todo not found")

// FindByID は指定されたIDのTodoタスクをデータベースから取得します。
func (r *Repository) FindByID(id int) (*Todo, error) {
	query := "SELECT id, title, completed, created_at FROM todos WHERE id = ?"

	var t Todo
	err := r.DB.QueryRow(query, id).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTodoNotFound
		}
		log.Printf("Failed to query todo by ID: %v", err)
		return nil, fmt.Errorf("could not query todo: %w", err)
	}

	return &t, nil
}

// Update は指定されたIDのTodoタスクを更新します。
func (r *Repository) Update(id int, t *Todo) (*Todo, error) {
	query := "UPDATE todos SET title = ?, completed = ? WHERE id = ?"

	result, err := r.DB.Exec(query, t.Title, t.Completed, id)
	if err != nil {
		log.Printf("Failed to update todo: %v", err)
		return nil, fmt.Errorf("could not update todo: %w", err)
	}

	// 更新された行数を確認
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("could not get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrTodoNotFound
	}

	// 更新されたTODOを取得して返す
	return r.FindByID(id)
}

// Delete は指定されたIDのTodoタスクを削除します。
func (r *Repository) Delete(id int) error {
	query := "DELETE FROM todos WHERE id = ?"

	result, err := r.DB.Exec(query, id)
	if err != nil {
		log.Printf("Failed to delete todo: %v", err)
		return fmt.Errorf("could not delete todo: %w", err)
	}

	// 削除された行数を確認
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTodoNotFound
	}

	return nil
}
