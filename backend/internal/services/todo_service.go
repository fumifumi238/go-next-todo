package services

import (
	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
)

// TodoService はTodo関連のビジネスロジックを扱います。
type TodoService struct {
	todoRepo *repositories.TodoRepository
}

// NewTodoService は新しいTodoServiceを作成します。
func NewTodoService(todoRepo *repositories.TodoRepository) *TodoService {
	return &TodoService{todoRepo: todoRepo}
}

// CreateTodo は新しいTodoを作成します。
func (s *TodoService) CreateTodo(todo *models.Todo, userID int) (*models.Todo, error) {
	todo.UserID = userID
	return s.todoRepo.Create(todo)
}

// GetTodos はユーザーのTodoを取得します。adminの場合は全Todo。
func (s *TodoService) GetTodos(userID int, userRole string) ([]*models.Todo, error) {
	if userRole == "admin" {
		return s.todoRepo.FindAll()
	}
	return s.todoRepo.FindByUserID(userID)
}

// GetTodoByID は指定IDのTodoを取得し、認可チェックを行います。
func (s *TodoService) GetTodoByID(id, userID int, userRole string) (*models.Todo, error) {
	todo, err := s.todoRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if todo.UserID != userID && userRole != "admin" {
		return nil, repositories.ErrTodoNotFound // アクセス拒否
	}
	return todo, nil
}

// UpdateTodo はTodoを更新し、認可チェックを行います。
func (s *TodoService) UpdateTodo(id int, updateTodo *models.Todo, userID int, userRole string) (*models.Todo, error) {
	existingTodo, err := s.todoRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if existingTodo.UserID != userID && userRole != "admin" {
		return nil, repositories.ErrTodoNotFound
	}
	updateTodo.UserID = existingTodo.UserID // 元の所有者を保持
	return s.todoRepo.Update(id, updateTodo)
}

// DeleteTodo はTodoを削除し、認可チェックを行います。
func (s *TodoService) DeleteTodo(id, userID int, userRole string) error {
	existingTodo, err := s.todoRepo.FindByID(id)
	if err != nil {
		return err
	}
	if existingTodo.UserID != userID && userRole != "admin" {
		return repositories.ErrTodoNotFound
	}
	return s.todoRepo.Delete(id)
}
