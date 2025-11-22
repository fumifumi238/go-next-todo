import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TodoList from '../TodoList';
import * as api from '../../../lib/api/todo';
import { Todo } from '../../../lib/types/todo';

// API関数をモック化
jest.mock('../../../lib/api/todo');

describe('TodoList', () => {
  const mockOnUpdate = jest.fn();

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

  it('TODOが空の場合はメッセージを表示する', () => {
    render(<TodoList todos={[]} onUpdate={mockOnUpdate} />);

    expect(screen.getByText('TODOがありません。新しいTODOを追加してください。')).toBeInTheDocument();
  });

  it('TODO一覧が正しく表示される', () => {
    render(<TodoList todos={mockTodos} onUpdate={mockOnUpdate} />);

    expect(screen.getByText('テストTODO 1')).toBeInTheDocument();
    expect(screen.getByText('テストTODO 2')).toBeInTheDocument();
  });

  it('完了済みTODOは取り消し線が表示される', () => {
    render(<TodoList todos={mockTodos} onUpdate={mockOnUpdate} />);

    const completedTodo = screen.getByText('テストTODO 2');
    expect(completedTodo).toHaveClass('line-through');
  });

  it('チェックボックスをクリックするとTODOの完了状態が切り替わる', async () => {
    const user = userEvent.setup();
    const mockUpdateTodo = api.updateTodo as jest.MockedFunction<typeof api.updateTodo>;
    mockUpdateTodo.mockResolvedValue({
      ...mockTodos[0],
      completed: true,
    });

    render(<TodoList todos={mockTodos} onUpdate={mockOnUpdate} />);

    const checkboxes = screen.getAllByRole('checkbox');
    const firstCheckbox = checkboxes[0];

    expect(firstCheckbox).not.toBeChecked();
    await user.click(firstCheckbox);

    await waitFor(() => {
      expect(mockUpdateTodo).toHaveBeenCalledWith(1, {
        title: 'テストTODO 1',
        completed: true,
      });
    });

    await waitFor(() => {
      expect(mockOnUpdate).toHaveBeenCalled();
    });
  });

  it('削除ボタンをクリックすると確認ダイアログが表示される', async () => {
    const user = userEvent.setup();
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(false);
    const mockDeleteTodo = api.deleteTodo as jest.MockedFunction<typeof api.deleteTodo>;

    render(<TodoList todos={mockTodos} onUpdate={mockOnUpdate} />);

    const deleteButtons = screen.getAllByRole('button', { name: '削除' });
    await user.click(deleteButtons[0]);

    expect(confirmSpy).toHaveBeenCalledWith('このTODOを削除しますか？');
    expect(mockDeleteTodo).not.toHaveBeenCalled();

    confirmSpy.mockRestore();
  });

  it('削除確認でOKを選択するとTODOが削除される', async () => {
    const user = userEvent.setup();
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
    const mockDeleteTodo = api.deleteTodo as jest.MockedFunction<typeof api.deleteTodo>;
    mockDeleteTodo.mockResolvedValue();

    render(<TodoList todos={mockTodos} onUpdate={mockOnUpdate} />);

    const deleteButtons = screen.getAllByRole('button', { name: '削除' });
    await user.click(deleteButtons[0]);

    await waitFor(() => {
      expect(mockDeleteTodo).toHaveBeenCalledWith(1);
    });

    await waitFor(() => {
      expect(mockOnUpdate).toHaveBeenCalled();
    });

    confirmSpy.mockRestore();
  });

  it('削除に失敗するとアラートが表示される', async () => {
    const user = userEvent.setup();
    const alertSpy = jest.spyOn(window, 'alert').mockImplementation(() => {});
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
    const mockDeleteTodo = api.deleteTodo as jest.MockedFunction<typeof api.deleteTodo>;
    mockDeleteTodo.mockRejectedValue(new Error('削除に失敗しました'));

    render(<TodoList todos={mockTodos} onUpdate={mockOnUpdate} />);

    const deleteButtons = screen.getAllByRole('button', { name: '削除' });
    await user.click(deleteButtons[0]);

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('TODOの削除に失敗しました');
    });

    expect(mockOnUpdate).not.toHaveBeenCalled();

    alertSpy.mockRestore();
    confirmSpy.mockRestore();
  });

  it('更新に失敗するとアラートが表示される', async () => {
    const user = userEvent.setup();
    const alertSpy = jest.spyOn(window, 'alert').mockImplementation(() => {});
    const mockUpdateTodo = api.updateTodo as jest.MockedFunction<typeof api.updateTodo>;
    mockUpdateTodo.mockRejectedValue(new Error('更新に失敗しました'));

    render(<TodoList todos={mockTodos} onUpdate={mockOnUpdate} />);

    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[0]);

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith('TODOの更新に失敗しました');
    });

    expect(mockOnUpdate).not.toHaveBeenCalled();

    alertSpy.mockRestore();
  });
});
