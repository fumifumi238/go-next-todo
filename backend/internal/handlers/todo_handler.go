package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-next-todo/backend/internal/models"
	"go-next-todo/backend/internal/repositories"
	"go-next-todo/backend/internal/services"
)

// TodoHandler はTodo関連のハンドラーを管理します。
type TodoHandler struct {
	todoService *services.TodoService
}

// NewTodoHandler は新しいTodoHandlerを作成します。
func NewTodoHandler(todoService *services.TodoService) *TodoHandler {
	return &TodoHandler{todoService: todoService}
}

// CreateTodoHandler は新しいTodoを作成します。
func (h *TodoHandler) CreateTodoHandler(c *gin.Context) {
	var newTodo models.Todo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	createdTodo, err := h.todoService.CreateTodo(&newTodo, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save todo to database"})
		return
	}
	c.JSON(http.StatusCreated, createdTodo)
}

// UpdateTodoHandler はTodoを更新します。
func (h *TodoHandler) UpdateTodoHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	var updateTodo models.Todo
	if err := c.ShouldBindJSON(&updateTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	updatedTodo, err := h.todoService.UpdateTodo(id, &updateTodo, userID, userRole)
	if err != nil {
		if err == repositories.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		if err == repositories.ErrTodoForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update todo"})
		return
	}
	c.JSON(http.StatusOK, updatedTodo)
}

// DeleteTodoHandler はTodoを削除します。
func (h *TodoHandler) DeleteTodoHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	err = h.todoService.DeleteTodo(id, userID, userRole)
	if err != nil {
		if err == repositories.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		if err == repositories.ErrTodoForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete todo"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetTodosHandler はTodoリストを取得します。
func (h *TodoHandler) GetTodosHandler(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	todos, err := h.todoService.GetTodos(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todos"})
		return
	}
	c.JSON(http.StatusOK, todos)
}

// GetTodoByIDHandler は指定IDのTodoを取得します。
func (h *TodoHandler) GetTodoByIDHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	todo, err := h.todoService.GetTodoByID(id, userID, userRole)
	if err != nil {
		if err == repositories.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		if err == repositories.ErrTodoForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todo"})
		return
	}
	c.JSON(http.StatusOK, todo)
}

// FindAllTodosAdminHandler はすべてのTodoを取得します（管理者専用）。
func (h *TodoHandler) FindAllTodosAdminHandler(c *gin.Context) {
	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: admin role required"})
		return
	}

	todos, err := h.todoService.GetAllTodos(userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todos"})
		return
	}
	c.JSON(http.StatusOK, todos)
}

// DeleteTodoAdminHandler は指定IDのTodoを削除します（管理者専用）。
func (h *TodoHandler) DeleteTodoAdminHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: admin role required"})
		return
	}

	err = h.todoService.DeleteTodo(id, 0, userRole) // userIDはadminの場合無視される
	if err != nil {
		if err == repositories.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete todo"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetTodosByUserIDHandler は指定ユーザーのTodoを取得します（管理者専用）。
func (h *TodoHandler) GetTodosByUserIDHandler(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	userRoleVal, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User role not found in context"})
		return
	}
	userRole, ok := userRoleVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user role type in context"})
		return
	}

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: admin role required"})
		return
	}

	todos, err := h.todoService.GetTodosByUserID(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch todos"})
		return
	}
	c.JSON(http.StatusOK, todos)
}
