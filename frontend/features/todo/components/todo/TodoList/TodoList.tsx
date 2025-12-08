"use client";

import { Todo } from "@/app/types/todo";
import { updateTodo, deleteTodo } from "@/lib/api/todo";

type TodoListProps = {
  todos: Todo[];
  onUpdate: () => void;
  token: string;
};

export default function TodoList({ todos, onUpdate, token }: TodoListProps) {
  const handleToggle = async (todo: Todo) => {
    if (todo.id === undefined) return;

    try {
      await updateTodo(
        todo.id,
        {
          title: todo.title,
          completed: !todo.completed,
          user_id: 1,
        },
        token
      );
      onUpdate();
    } catch (error) {
      console.error("Failed to update todo:", error);
      alert("TODOの更新に失敗しました");
    }
  };

  const handleDelete = async (id: number | undefined) => {
    if (id === undefined) {
      alert("IDが未指定です");
      return;
    }
    if (!confirm("このTODOを削除しますか？")) return;

    try {
      await deleteTodo(id, token);
      onUpdate();
    } catch (error) {
      console.error("Failed to delete todo:", error);
      alert("TODOの削除に失敗しました");
    }
  };

  if (todos.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <p>TODOがありません。新しいTODOを追加してください。</p>
      </div>
    );
  }

  return (
    <ul className="space-y-2">
      {todos.map((todo, index) => (
        <li
          key={index}
          className="bg-card border border-border rounded-lg p-4 flex items-center gap-4 hover:shadow-sm transition-shadow">
          <input
            type="checkbox"
            checked={todo.completed}
            onChange={() => handleToggle(todo)}
            className="w-5 h-5 rounded border--border text-primary focus:ring-2 focus:ring-ring cursor-pointer"
          />
          <span
            className={`flex-1 ${
              todo.completed
                ? "line-through text-muted-foreground"
                : "text-foreground"
            }`}>
            {todo.title}
          </span>
          <button
            onClick={() => handleDelete(todo.id)}
            className="px-3 py-1 text-sm bg-red-500 hover:bg-red-600 text-white rounded transition-colors">
            削除
          </button>
        </li>
      ))}
    </ul>
  );
}
