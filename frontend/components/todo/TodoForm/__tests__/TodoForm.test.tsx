import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import TodoForm from "@/components/todo/TodoForm/TodoForm";
import * as api from "@/lib/api/todo";
import { Todo } from "@/app/types/todo";

// API関数をモック化
jest.mock("@/lib/api/todo");

describe("TodoForm", () => {
  const mockOnAdd = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("フォームが正しくレンダリングされる", () => {
    render(<TodoForm onAdd={mockOnAdd} token="fake-token" />);

    const input = screen.getByPlaceholderText("新しいTODOを入力...");
    const button = screen.getByRole("button", { name: "追加" });

    expect(input).toBeInTheDocument();
    expect(button).toBeInTheDocument();
    expect(button).toBeDisabled(); // 初期状態では空なので無効
  });

  it("入力フィールドにテキストを入力できる", async () => {
    const user = userEvent.setup();
    render(<TodoForm onAdd={mockOnAdd} token="fake-token" />);

    const input = screen.getByPlaceholderText(
      "新しいTODOを入力..."
    ) as HTMLInputElement;
    await user.type(input, "新しいTODO");

    expect(input.value).toBe("新しいTODO");
  });

  it("テキストを入力するとボタンが有効になる", async () => {
    const user = userEvent.setup();
    render(<TodoForm onAdd={mockOnAdd} token="fake-token" />);

    const input = screen.getByPlaceholderText("新しいTODOを入力...");
    const button = screen.getByRole("button", { name: "追加" });

    expect(button).toBeDisabled();
    await user.type(input, "新しいTODO");
    expect(button).toBeEnabled();
  });

  it("空のテキストで送信するとアラートが表示される", async () => {
    const alertSpy = jest.spyOn(window, "alert").mockImplementation(() => {});

    // フォームを直接送信（ボタンは無効なので、フォームのsubmitイベントを直接発火）
    const { container } = render(
      <TodoForm onAdd={mockOnAdd} token="fake-token" />
    );
    const form = container.querySelector("form");

    if (form) {
      form.dispatchEvent(
        new Event("submit", { cancelable: true, bubbles: true })
      );
    }

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("TODOのタイトルを入力してください");
    });

    expect(mockOnAdd).not.toHaveBeenCalled();
    alertSpy.mockRestore();
  });

  it("TODOを作成できる", async () => {
    const user = userEvent.setup();
    const mockCreateTodo = api.createTodo as jest.MockedFunction<
      typeof api.createTodo
    >;
    mockCreateTodo.mockResolvedValue({
      id: 1,
      title: "新しいTODO",
      completed: false,
      created_at: new Date().toISOString(),
      user_id: 1,
    });

    render(<TodoForm onAdd={mockOnAdd} token="fake-token" />);

    const input = screen.getByPlaceholderText("新しいTODOを入力...");
    const button = screen.getByRole("button", { name: "追加" });

    await user.type(input, "新しいTODO");
    await user.click(button);

    await waitFor(() => {
      expect(mockCreateTodo).toHaveBeenCalledWith(
        {
          title: "新しいTODO",
          completed: false,
          user_id: 1,
        },
        "fake-token"
      );
    });

    await waitFor(() => {
      expect(mockOnAdd).toHaveBeenCalled();
    });

    // 入力フィールドがクリアされる
    expect(input).toHaveValue("");
  });

  it("送信中はボタンが「追加中...」と表示され、無効になる", async () => {
    const user = userEvent.setup();
    const mockCreateTodo = api.createTodo as jest.MockedFunction<
      typeof api.createTodo
    >;

    // 解決を遅延させる
    let resolvePromise: (value: Todo) => void;
    const promise = new Promise<Todo>((resolve) => {
      resolvePromise = resolve;
    });
    mockCreateTodo.mockReturnValue(promise);

    render(<TodoForm onAdd={mockOnAdd} token="fake-token" />);

    const input = screen.getByPlaceholderText("新しいTODOを入力...");
    const button = screen.getByRole("button", { name: "追加" });

    await user.type(input, "新しいTODO");
    await user.click(button);

    // 送信中は「追加中...」と表示され、無効
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "追加中..." })
      ).toBeInTheDocument();
    });
    expect(input).toBeDisabled();

    // 完了させる
    resolvePromise!({
      id: 1,
      title: "新しいTODO",
      completed: false,
      created_at: new Date().toISOString(),
      user_id: 1,
    });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "追加" })).toBeInTheDocument();
    });
  });

  it("TODO作成に失敗するとアラートが表示される", async () => {
    const user = userEvent.setup();
    const alertSpy = jest.spyOn(window, "alert").mockImplementation(() => {});
    const mockCreateTodo = api.createTodo as jest.MockedFunction<
      typeof api.createTodo
    >;
    mockCreateTodo.mockRejectedValue(new Error("作成に失敗しました"));

    render(<TodoForm onAdd={mockOnAdd} token="fake-token" />);

    const input = screen.getByPlaceholderText("新しいTODOを入力...");
    const button = screen.getByRole("button", { name: "追加" });

    await user.type(input, "新しいTODO");
    await user.click(button);

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("TODOの作成に失敗しました");
    });

    expect(mockOnAdd).not.toHaveBeenCalled();

    alertSpy.mockRestore();
  });
});
