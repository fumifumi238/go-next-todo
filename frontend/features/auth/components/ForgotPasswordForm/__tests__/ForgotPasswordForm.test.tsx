import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ForgotPasswordForm from "@/features/auth/components/ForgotPasswordForm/ForgotPasswordForm";
import * as api from "@/lib/api";

// apiモジュールをモック
jest.mock("@/lib/api", () => ({
  forgotPassword: jest.fn(),
}));

describe("ForgotPasswordForm", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(window, "alert").mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("フォームが正しくレンダリングされること", () => {
    render(<ForgotPasswordForm />);
    expect(screen.getByLabelText(/メールアドレス/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /送信/i })).toBeInTheDocument();
  });

  it("有効なメールアドレスで送信に成功すること", async () => {
    const mockForgotPassword = api.forgotPassword as jest.Mock;
    mockForgotPassword.mockResolvedValue({
      data: { message: "パスワードリセットのメールを送信しました" },
    });

    render(<ForgotPasswordForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "test@example.com"
    );
    await userEvent.click(screen.getByRole("button", { name: /送信/i }));

    await waitFor(() => {
      expect(mockForgotPassword).toHaveBeenCalledWith("test@example.com");
      expect(window.alert).toHaveBeenCalledWith(
        "パスワードリセットのメールを送信しました。メールを確認してください。"
      );
    });
  });

  it("無効なメールアドレスでバリデーションエラーが表示されること", async () => {
    render(<ForgotPasswordForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "invalid-email"
    );
    await userEvent.click(screen.getByRole("button", { name: /送信/i }));

    await waitFor(() => {
      expect(
        screen.getByText("有効なメールアドレスを入力してください")
      ).toBeInTheDocument();
    });
    expect(api.forgotPassword).not.toHaveBeenCalled();
  });

  it("APIエラーが発生した場合、エラーメッセージを表示すること", async () => {
    const mockForgotPassword = api.forgotPassword as jest.Mock;
    mockForgotPassword.mockResolvedValue({ error: "ユーザーが見つかりません" });

    render(<ForgotPasswordForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "notfound@example.com"
    );
    await userEvent.click(screen.getByRole("button", { name: /送信/i }));

    await waitFor(() => {
      expect(screen.getByText("ユーザーが見つかりません")).toBeInTheDocument();
    });
  });
});
