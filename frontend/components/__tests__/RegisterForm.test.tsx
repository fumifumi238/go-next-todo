import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import RegisterForm from "@/components/RegisterForm";
import * as api from "@/lib/api"; // apiモジュールをモックするためにインポート

// apiモジュール全体をモック
jest.mock("@/lib/api", () => ({
  registerUser: jest.fn(),
}));

describe("RegisterForm", () => {
  // 各テストの前にモックをリセット
  beforeEach(() => {
    jest.spyOn(window, "alert").mockImplementation(() => {});
    jest.clearAllMocks(); // 追加: すべてのモックの呼び出し履歴をクリア
  });

  afterEach(() => {
    jest.restoreAllMocks(); // 各テスト後にモックをクリーンアップ
  });

  it("フォームが正しくレンダリングされること", () => {
    render(<RegisterForm />);
    expect(screen.getByLabelText(/ユーザー名/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/メールアドレス/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/パスワード/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /登録/i })).toBeInTheDocument();
  });

  it("すべてのフィールドが入力されたときに登録に成功すること", async () => {
    const mockRegisterUser = api.registerUser as jest.Mock;
    mockRegisterUser.mockResolvedValueOnce({
      data: { message: "ユーザー登録が成功しました！", user_id: 1 },
    });

    render(<RegisterForm />);

    await userEvent.type(screen.getByLabelText(/ユーザー名/i), "testuser");
    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "test@example.com"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "password123");

    await userEvent.click(screen.getByRole("button", { name: /登録/i }));

    await waitFor(() => {
      expect(mockRegisterUser).toHaveBeenCalledWith({
        username: "testuser",
        email: "test@example.com",
        password: "password123",
      });
    });

    // 成功メッセージが表示される（alertはテストできないため、API呼び出しとフォームリセットを確認）
    expect(screen.getByLabelText(/ユーザー名/i)).toHaveValue(""); // フォームがリセットされていること
  });

  it("必須フィールドが空の場合、バリデーションエラーを表示すること", async () => {
    render(<RegisterForm />);

    const usernameInput = screen.getByLabelText(/ユーザー名/i);
    const emailInput = screen.getByLabelText(/メールアドレス/i);
    const passwordInput = screen.getByLabelText(/パスワード/i);

    // 各フィールドを明示的にクリア
    await userEvent.clear(usernameInput);
    await userEvent.clear(emailInput);
    await userEvent.clear(passwordInput);

    // 各フィールドからフォーカスを外してバリデーションをトリガー (mode: 'onBlur' の場合を考慮)
    await userEvent.tab(); // usernameからフォーカスアウト
    await userEvent.tab(); // emailからフォーカスアウト
    await userEvent.tab(); // passwordからフォーカスアウト

    await userEvent.click(screen.getByRole("button", { name: /登録/i }));

    await waitFor(() => {
      expect(
        screen.getByText(/ユーザー名は3文字以上である必要があります/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/有効なメールアドレスを入力してください/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/パスワードは8文字以上である必要があります/i)
      ).toBeInTheDocument();
    });
    expect(api.registerUser).not.toHaveBeenCalled(); // APIが呼び出されていないこと
  });

  it("メールアドレスが無効な場合、バリデーションエラーを表示すること", async () => {
    render(<RegisterForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "invalid-email"
    );
    await userEvent.click(screen.getByRole("button", { name: /登録/i }));

    // waitFor を使用して、エラーメッセージが DOM に現れるまで待機
    await waitFor(() => {
      expect(
        screen.getByText("有効なメールアドレスを入力してください")
      ).toBeInTheDocument();
    });
  });

  it("APIエラーが発生した場合、エラーメッセージを表示すること", async () => {
    const mockRegisterUser = api.registerUser as jest.Mock;
    mockRegisterUser.mockResolvedValueOnce({
      error: "Username or email already exists",
    });

    render(<RegisterForm />);

    await userEvent.type(screen.getByLabelText(/ユーザー名/i), "duplicate");
    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "duplicate@example.com"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "password123");

    await userEvent.click(screen.getByRole("button", { name: /登録/i }));

    // 修正: `queryByText` や `getAllByText` ではなく、単一のエラーメッセージを期待
    await waitFor(() => {
      expect(
        screen.getByText("Username or email already exists")
      ).toBeInTheDocument();
    });
  });
});
