'use client';

import { useState, useEffect } from 'react';
import { Todo } from './lib/types/todo';
import { fetchTodos } from './lib/api/todo';
import TodoForm from './components/todo/TodoForm';
import TodoList from './components/todo/TodoList';



export default function Page() {


  const [todos, setTodos] = useState<Todo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTodos = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await fetchTodos();
      setTodos(data);
          console.log('Fetched Todos Data:', data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'TODOの読み込みに失敗しました');
      console.error('Failed to load todos:', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadTodos();
  }, []);

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-2xl mx-auto">
        <div className="bg-[var(--card)] border border-[var(--border)] rounded-lg p-6 shadow-sm">
          <h1 className="text-3xl font-bold mb-6 text-[var(--foreground)]">
            TODOアプリ
          </h1>

          <TodoForm onAdd={loadTodos} />

          {error && (
            <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-300 dark:border-red-700 text-red-800 dark:text-red-300 rounded-lg">
              <p className="font-semibold mb-2">エラー</p>
              <p className="mb-3">{error}</p>
              <button
                onClick={loadTodos}
                className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors"
              >
                再試行
              </button>
            </div>
          )}

          {isLoading ? (
            <div className="text-center py-8 text-[var(--muted-foreground)]">
              <p>読み込み中...</p>
            </div>
          ) : (
            <TodoList todos={todos} onUpdate={loadTodos} />
          )}

          {!isLoading && !error && todos.length > 0 && (
            <div className="mt-4 text-sm text-[var(--muted-foreground)]">
              <p>
                合計: {todos.length}件 / 完了: {todos.filter((t) => t.completed).length}件 / 未完了:{' '}
                {todos.filter((t) => !t.completed).length}件
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
