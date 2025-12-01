'use client';

import { useState } from 'react';
import { createTodo } from '@/lib/api/todo';

interface TodoFormProps {
  onAdd: () => void;
}

export default function TodoForm({ onAdd }: TodoFormProps) {
  const [title, setTitle] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // title.trim() === ''のほうが読みやすいです。
    if (title.trim()==='') {
      alert('TODOのタイトルを入力してください');
      return;
    }

    setIsSubmitting(true);
    try {
      await createTodo({
        title: title.trim(),
        completed: false,
        user_id:1 // 仮設定
      });
      setTitle('');
      onAdd();
    } catch (error) {
      console.error('Failed to create todo:', error);
      alert('TODOの作成に失敗しました');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="mb-6">
      <div className="flex gap-2">
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="新しいTODOを入力..."
          disabled={isSubmitting}
          className="flex-1 px-4 py-2 border border-[var(--border)] rounded-lg bg-[var(--card)] text-[var(--foreground)] placeholder-[var(--muted-foreground)] focus:outline-none focus:ring-2 focus:ring-[var(--ring)] disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={isSubmitting || title.trim() ===''}
          className="px-6 py-2 bg-[var(--primary)] text-[var(--primary-foreground)] rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isSubmitting ? '追加中...' : '追加'}
        </button>
      </div>
    </form>
  );
}
