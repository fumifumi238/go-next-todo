"use client";

import { useState, useEffect } from "react";
import { Todo } from "@/app/types/todo";
import { fetchTodos } from "@/lib/api/todo";
import TodoForm from "@/features/todo/components/todo/TodoForm/TodoForm";
import TodoList from "@/features/todo/components/todo/TodoList/TodoList";
import { useRouter } from "next/navigation"; // useRouterをインポート
import { AuthContext } from "@/context/AuthContext"; // AuthContextをインポート
import { useContext, useCallback } from "react";

export default function Page() {
  const [todos, setTodos] = useState<Todo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { token, logout } = useContext(AuthContext);
  const router = useRouter();

  useEffect(() => {
    if (!token) {
      router.push("/login");
    }
  }, [token, router]);

  const loadTodos = useCallback(async () => {
    if (!token) {
      // トークンがない場合はAPIを呼び出さない
      setIsLoading(false);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      const data = await fetchTodos(token);
      setTodos(data);
      console.log("Fetched Todos Data:", data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "TODOの読み込みに失敗しました"
      );
      console.error("Failed to load todos:", err);
    } finally {
      setIsLoading(false);
    }
  }, [token]);

  useEffect(() => {
    loadTodos();
  }, [loadTodos]);
  if (!token) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <p>リダイレクト中...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-2xl mx-auto">
        <div className="bg-card border border-border rounded-lg p-6 shadow-sm">
          <h1 className="text-3xl font-bold mb-6 text-foreground">
            TODOアプリ
          </h1>
          <button
            onClick={logout}
            className="mb-4 px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors">
            ログアウト
          </button>
          {token && <TodoForm onAdd={loadTodos} token={token} />}{" "}
          {/* tokenを渡す */}
          {error && (
            <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-300 dark:border-red-700 text-red-800 dark:text-red-300 rounded-lg">
              <p className="font-semibold mb-2">エラー</p>
              <p className="mb-3">{error}</p>
              <button
                onClick={loadTodos}
                className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors">
                再試行
              </button>
            </div>
          )}
          {isLoading ? (
            <div className="text-center py-8 text-muted-foreground">
              <p>読み込み中...</p>
            </div>
          ) : (
            token && (
              <TodoList todos={todos} onUpdate={loadTodos} token={token} />
            ) // tokenを渡す
          )}
          {!isLoading && !error && todos.length > 0 && (
            <div className="mt-4 text-sm text-muted-foreground">
              <p>
                合計: {todos.length}件 / 完了:{" "}
                {todos.filter((t) => t.completed).length}件 / 未完了:{" "}
                {todos.filter((t) => !t.completed).length}件
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
