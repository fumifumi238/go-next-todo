import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Page from '../page';
import * as api from '../lib/api/todo';
import { Todo } from '../lib/types/todo';

// API関数をモック化
jest.mock('../lib/api/todo');

describe('Page', () => {
  const mockTodos: Todo[] = [
    {
      id: 1,
      title: 'テストTODO 1',
      completed: false,
      created_at: new Date().toISOString(),
    },
    {
      id: 2,
      title: 'テストTODO 2',
      completed: true,
      created_at: new Date().toISOString(),
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('ローディング中は「読み込み中...」と表示される', async () => {
    const mockFetchTodos = api.fetchTodos as jest.MockedFunction<typeof api.fetchTodos>;

    // 解決を遅延させる
    let resolvePromise: (value: Todo[]) => void;
    const promise = new Promise<Todo[]>((resolve) => {
      resolvePromise = resolve;
    });
    mockFetchTodos.mockReturnValue(promise);

    render(<Page />);

    expect(screen.getByText('読み込み中...')).toBeInTheDocument();

    // 完了させる
    resolvePromise!(mockTodos);

    await waitFor(() => {
      expect(screen.queryByText('読み込み中...')).not.toBeInTheDocument();
    });
  });

  it('TODO一覧が正しく表示される', async () => {
    const mockFetchTodos = api.fetchTodos as jest.MockedFunction<typeof api.fetchTodos>;
    mockFetchTodos.mockResolvedValue(mockTodos);

    render(<Page />);

    await waitFor(() => {
      expect(screen.getByText('テストTODO 1')).toBeInTheDocument();
    });

    expect(screen.getByText('テストTODO 2')).toBeInTheDocument();
  });

  it('エラーが発生した場合はエラーメッセージが表示される', async () => {
    const mockFetchTodos = api.fetchTodos as jest.MockedFunction<typeof api.fetchTodos>;
    mockFetchTodos.mockRejectedValue(new Error('サーバーエラー'));

    render(<Page />);

    await waitFor(() => {
      expect(screen.getByText('エラー')).toBeInTheDocument();
    });

    expect(screen.getByText('サーバーエラー')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '再試行' })).toBeInTheDocument();
  });

  it('統計情報が正しく表示される', async () => {
    const mockFetchTodos = api.fetchTodos as jest.MockedFunction<typeof api.fetchTodos>;
    mockFetchTodos.mockResolvedValue(mockTodos);

    render(<Page />);

    await waitFor(() => {
      expect(screen.getByText(/合計: 2件/)).toBeInTheDocument();
    });

    expect(screen.getByText(/完了: 1件/)).toBeInTheDocument();
    expect(screen.getByText(/未完了: 1件/)).toBeInTheDocument();
  });

  it('再試行ボタンをクリックすると再度データを取得する', async () => {
    const user = userEvent.setup();
    const mockFetchTodos = api.fetchTodos as jest.MockedFunction<typeof api.fetchTodos>;

    // 最初はエラー、次は成功
    mockFetchTodos
      .mockRejectedValueOnce(new Error('サーバーエラー'))
      .mockResolvedValueOnce(mockTodos);

    render(<Page />);

    // エラーが表示されるまで待つ
    await waitFor(() => {
      expect(screen.getByText('エラー')).toBeInTheDocument();
    });

    // 再試行ボタンをクリック
    const retryButton = screen.getByRole('button', { name: '再試行' });
    await user.click(retryButton);

    // TODO一覧が表示される（ローディングは一瞬なので、直接結果を待つ）
    await waitFor(() => {
      expect(screen.getByText('テストTODO 1')).toBeInTheDocument();
    }, { timeout: 3000 });

    expect(mockFetchTodos).toHaveBeenCalledTimes(2);
  });
});
