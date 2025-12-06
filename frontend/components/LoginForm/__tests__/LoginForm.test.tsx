// frontend/components/LoginForm/__tests__/LoginForm.test.tsx
import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import LoginForm from "@/components/LoginForm/LoginForm";
import * as api from "@/lib/api";
import { AuthContext } from "@/context/AuthContext";

// next/navigation をモック
const mockPush = jest.fn();
jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

// apiモジュール全体をモック
jest.mock("@/lib/api", () => ({
  loginUser: jest.fn(),
}));

describe("LoginForm", () => {
  const mockLogin = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(window, "alert").mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  // AuthContext Providerのヘルパーコンポーネント
  const renderWithAuthContext = (ui: React.ReactElement) => {
    return render(
      <AuthContext.Provider
        value={{ token: null, login: mockLogin, logout: jest.fn() }}>
        {ui}
      </AuthContext.Provider>
    );
  };

  it("フォームが正しくレンダリングされること", () => {
    renderWithAuthContext(<LoginForm />);
    expect(screen.getByLabelText(/メールアドレス/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/パスワード/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /ログイン/i })
    ).toBeInTheDocument();
  });

  it("有効なクレデンシャルでログインに成功すること", async () => {
    const mockLoginUser = api.loginUser as jest.Mock;
    mockLoginUser.mockResolvedValueOnce({
      data: { token: "fake-jwt-token", user_id: 1, role: "user" },
    });

    renderWithAuthContext(<LoginForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "test@example.com"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "password123");

    await userEvent.click(screen.getByRole("button", { name: /ログイン/i }));

    await waitFor(() => {
      expect(mockLoginUser).toHaveBeenCalledWith({
        email: "test@example.com",
        password: "password123",
      });
      expect(mockLogin).toHaveBeenCalledWith("fake-jwt-token"); // AuthContextのloginが呼ばれる
      expect(window.alert).toHaveBeenCalledWith("ログインに成功しました！");
      expect(mockPush).toHaveBeenCalledWith("/"); // ルートページにリダイレクト
    });

    // フォームがリセットされていることを確認
    expect(screen.getByLabelText(/メールアドレス/i)).toHaveValue("");
    expect(screen.getByLabelText(/パスワード/i)).toHaveValue("");
  });

  it("必須フィールドが空の場合、ログインボタンが無効になること", async () => {
    renderWithAuthContext(<LoginForm />);

    const emailInput = screen.getByLabelText(/メールアドレス/i);
    const passwordInput = screen.getByLabelText(/パスワード/i);
    const loginButton = screen.getByRole("button", { name: /ログイン/i });

    // 初期状態ではボタンは有効（フォームがリセットされている場合）
    // expect(loginButton).toBeEnabled(); // 必要であれば

    await userEvent.clear(emailInput);
    await userEvent.clear(passwordInput);

    await userEvent.tab(); // フォーカスを外してバリデーションをトリガー

    await waitFor(() => {
      expect(loginButton).toBeDisabled(); // ボタンが無効になっていることを確認
    });

    expect(api.loginUser).not.toHaveBeenCalled();
  });

  it("メールアドレスが無効な場合、バリデーションエラーを表示すること", async () => {
    renderWithAuthContext(<LoginForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "invalid-email"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "password123");

    await userEvent.click(screen.getByRole("button", { name: /ログイン/i }));

    await waitFor(() => {
      expect(
        screen.getByText("有効なメールアドレスを入力してください")
      ).toBeInTheDocument();
    });
    expect(api.loginUser).not.toHaveBeenCalled(); // APIが呼び出されていないこと
  });

  it("APIエラーが発生した場合、エラーメッセージを表示すること", async () => {
    const mockLoginUser = api.loginUser as jest.Mock;
    mockLoginUser.mockResolvedValueOnce({
      error: "無効なメールアドレスまたはパスワードです。",
    });

    renderWithAuthContext(<LoginForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "wrong@example.com"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "wrongpassword");

    await userEvent.click(screen.getByRole("button", { name: /ログイン/i }));

    await waitFor(() => {
      expect(
        screen.getByText("無効なメールアドレスまたはパスワードです。")
      ).toBeInTheDocument();
    });
    expect(mockLogin).not.toHaveBeenCalled(); // ログインは呼ばれない
    expect(mockPush).not.toHaveBeenCalled(); // リダイレクトもされない
  });

  it("ネットワークエラーなどの予期せぬ例外が発生した場合、エラーメッセージを表示すること", async () => {
    const mockLoginUser = api.loginUser as jest.Mock;
    mockLoginUser.mockRejectedValueOnce(new Error("Network Error")); // ネットワークエラーをシミュレート

    renderWithAuthContext(<LoginForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "network@example.com"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "password123");

    await userEvent.click(screen.getByRole("button", { name: /ログイン/i }));

    await waitFor(() => {
      expect(
        screen.getByText("ネットワークエラーによりログインに失敗しました。")
      ).toBeInTheDocument();
    });
    expect(mockLogin).not.toHaveBeenCalled();
    expect(mockPush).not.toHaveBeenCalled();
  });
});
