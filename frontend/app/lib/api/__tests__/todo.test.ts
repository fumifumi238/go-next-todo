import { fetchTodos, createTodo, updateTodo, deleteTodo } from '../todo';
import { Todo } from '../../types/todo';

// fetchをモック化
global.fetch = jest.fn();

describe('API Functions', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('fetchTodos', () => {
    it('TODO一覧を取得できる', async () => {
      const mockTodos: Todo[] = [
        {
          id: 1,
          title: 'テストTODO 1',
          completed: false,
          created_at: new Date().toISOString(),
        },
      ];

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockTodos,
      });

      const result = await fetchTodos();

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/todos', {
        cache: 'no-store',
      });
      expect(result).toEqual(mockTodos);
    });

    it('エラーレスポンスの場合、エラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => ({
          error: 'データベースエラー',
          details: '接続に失敗しました',
        }),
      });

      await expect(fetchTodos()).rejects.toThrow('データベースエラー (接続に失敗しました)');
    });

    it('ネットワークエラーの場合、適切なエラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new TypeError('Failed to fetch'));

      await expect(fetchTodos()).rejects.toThrow(
        'バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。'
      );
    });
  });

  describe('createTodo', () => {
    it('TODOを作成できる', async () => {
      const newTodo = {
        title: '新しいTODO',
        completed: false,
      };

      const createdTodo: Todo = {
        id: 1,
        ...newTodo,
        created_at: new Date().toISOString(),
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => createdTodo,
      });

      const result = await createTodo(newTodo);

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/todos', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newTodo),
      });
      expect(result).toEqual(createdTodo);
    });

    it('エラーレスポンスの場合、エラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: async () => ({
          error: 'タイトルは必須です',
        }),
      });

      await expect(
        createTodo({ title: '', completed: false })
      ).rejects.toThrow('タイトルは必須です');
    });

    it('ネットワークエラーの場合、適切なエラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new TypeError('Failed to fetch'));

      await expect(
        createTodo({ title: '新しいTODO', completed: false })
      ).rejects.toThrow(
        'バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。'
      );
    });
  });

  describe('updateTodo', () => {
    it('TODOを更新できる', async () => {
      const updateData = {
        title: '更新されたTODO',
        completed: true,
      };

      const updatedTodo: Todo = {
        id: 1,
        ...updateData,
        created_at: new Date().toISOString(),
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => updatedTodo,
      });

      const result = await updateTodo(1, updateData);

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/todos/1', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateData),
      });
      expect(result).toEqual(updatedTodo);
    });

    it('エラーレスポンスの場合、エラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({
          error: 'TODOが見つかりません',
        }),
      });

      await expect(
        updateTodo(999, { title: '更新', completed: false })
      ).rejects.toThrow('TODOが見つかりません');
    });

    it('JSONパースに失敗した場合、デフォルトエラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => {
          throw new Error('Invalid JSON');
        },
      });

      await expect(
        updateTodo(1, { title: '更新', completed: false })
      ).rejects.toThrow('Failed to update todo: 500 Internal Server Error');
    });

    it('ネットワークエラーの場合、適切なエラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new TypeError('Failed to fetch'));

      await expect(
        updateTodo(1, { title: '更新', completed: false })
      ).rejects.toThrow(
        'バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。'
      );
    });
  });

  describe('deleteTodo', () => {
    it('TODOを削除できる', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
      });

      await deleteTodo(1);

      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/todos/1', {
        method: 'DELETE',
      });
    });

    it('エラーレスポンスの場合、エラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({
          error: 'TODOが見つかりません',
        }),
      });

      await expect(deleteTodo(999)).rejects.toThrow('TODOが見つかりません');
    });

    it('JSONパースに失敗した場合、デフォルトエラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => {
          throw new Error('Invalid JSON');
        },
      });

      await expect(deleteTodo(1)).rejects.toThrow('Failed to delete todo: 500 Internal Server Error');
    });

    it('ネットワークエラーの場合、適切なエラーメッセージを投げる', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new TypeError('Failed to fetch'));

      await expect(deleteTodo(1)).rejects.toThrow(
        'バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。'
      );
    });
  });
});
