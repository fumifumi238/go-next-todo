// Package repositories はデータベース操作を行うリポジトリを提供します。
package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"go-next-todo/backend/internal/models"
)

// TodoRepository はデータベース操作を行うための構造体です。
type TodoRepository struct {
	DB DataStore
}

// NewTodoRepository は新しいTodoRepositoryインスタンスを作成します。
func NewTodoRepository(db DataStore) *TodoRepository {
	return &TodoRepository{DB: db}
}

// ErrTodoNotFound はTODOが見つからない場合のエラーです。
var ErrTodoNotFound = errors.New("todo not found")

// ErrTodoForbidden はTODOへのアクセスが禁止されている場合のエラーです。
var ErrTodoForbidden = errors.New("todo access forbidden")

// Create は新しいTodoタスクをデータベースに挿入します。
func (r *TodoRepository) Create(t *models.Todo) (*models.Todo, error) {
	query := "INSERT INTO todos (user_id, title, completed) VALUES (?, ?, ?)" // 💡 user_id を追加

	result, err := r.DB.Exec(query, t.UserID, t.Title, t.Completed) // 💡 t.UserID を追加
	if err != nil {
		log.Printf("Failed to insert todo: %v", err)
		return nil, fmt.Errorf("could not insert todo: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert ID: %w", err)
	}

	// 💡 挿入されたTODOをDBから取得し直すことで、正確な created_at/updated_at を反映させる
	createdTodo, err := r.FindByID(int(id))
	if err != nil {
		return nil, fmt.Errorf("could not find created todo: %w", err)
	}

	return createdTodo, nil
}

// FindAll はすべてのTodoタスクをデータベースから取得します。
func (r *TodoRepository) FindAll() ([]*models.Todo, error) {
	query := "SELECT id, user_id, title, completed, created_at, updated_at FROM todos ORDER BY created_at DESC" // 💡 user_id, updated_at を追加

	rows, err := r.DB.Query(query)
	if err != nil {
		log.Printf("Failed to query todos: %v", err)
		return nil, fmt.Errorf("could not query todos: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		var t models.Todo
		err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt) // 💡 t.UserID, t.UpdatedAt を追加
		if err != nil {
			log.Printf("Failed to scan todo: %v", err)
			return nil, fmt.Errorf("could not scan todo: %w", err)
		}
		todos = append(todos, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating todos: %w", err)
	}

	// 💡 ここを修正: 結果が空の場合でも、nilではなく空のスライスを返す
	if todos == nil {
		return []*models.Todo{}, nil // 明示的に空のスライスを返す
	}

	return todos, nil
}

// FindByID は指定されたIDのTodoタスクをデータベースから取得します。
func (r *TodoRepository) FindByID(id int) (*models.Todo, error) {
	query := "SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE id = ?" // 💡 user_id, updated_at を追加

	var t models.Todo
	err := r.DB.QueryRow(query, id).Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt) // 💡 t.UserID, t.UpdatedAt を追加
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTodoNotFound
		}
		log.Printf("Failed to query todo by ID: %v", err)
		return nil, fmt.Errorf("could not query todo: %w", err)
	}

	return &t, nil
}

func (r *TodoRepository) FindByUserID(userID int) ([]*models.Todo, error) { // FindAllと同様にポインタのスライスを返すように変更
	rows, err := r.DB.Query("SELECT id, user_id, title, completed, created_at, updated_at FROM todos WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		log.Printf("Failed to query todos by user ID: %v", err)
		return nil, fmt.Errorf("could not query todos by user ID: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo // ポインタのスライス
	for rows.Next() {
		var t models.Todo
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			log.Printf("Failed to scan todo by user ID: %v", err)
			return nil, fmt.Errorf("could not scan todo by user ID: %w", err)
		}
		todos = append(todos, &t) // アドレスをappend
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating todos by user ID: %w", err)
	}

	// 結果が空の場合でも、nilではなく空のスライスを返す
	if todos == nil {
		return []*models.Todo{}, nil
	}

	return todos, nil
}

// Update は指定されたIDのTodoタスクを更新します。
func (r *TodoRepository) Update(id int, t *models.Todo) (*models.Todo, error) {
	query := "UPDATE todos SET title = ?, completed = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?" // 💡 updated_at を追加

	result, err := r.DB.Exec(query, t.Title, t.Completed, id)
	if err != nil {
		log.Printf("Failed to update todo: %v", err)
		return nil, fmt.Errorf("could not update todo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("could not get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrTodoNotFound
	}

	return r.FindByID(id)
}

// Delete は指定されたIDのTodoタスクを削除します。
func (r *TodoRepository) Delete(id int) error {
	query := "DELETE FROM todos WHERE id = ?"

	result, err := r.DB.Exec(query, id)
	if err != nil {
		log.Printf("Failed to delete todo: %v", err)
		return fmt.Errorf("could not delete todo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTodoNotFound
	}

	return nil
}
