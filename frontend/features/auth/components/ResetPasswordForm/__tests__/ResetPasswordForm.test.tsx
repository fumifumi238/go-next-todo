import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ResetPasswordForm from "@/features/auth/components/ResetPasswordForm/ResetPasswordForm";
import * as api from "@/lib/api";

// apiモジュールをモック
jest.mock("@/lib/api", () => ({
  resetPassword: jest.fn(),
}));

describe("ResetPasswordForm", () => {
  const mockToken = "test-token";

  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(window, "alert").mockImplementation(() => {});
    // location.href をモック
    Object.defineProperty(window, "location", {
      value: { href: "" },
      writable: true,
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("フォームが正しくレンダリングされること", () => {
    render(<ResetPasswordForm token={mockToken} />);
    expect(screen.getByLabelText(/新しいパスワード/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/パスワード確認/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /パスワードをリセット/i })
    ).toBeInTheDocument();
  });

  it("有効なパスワードで送信に成功すること", async () => {
    const mockResetPassword = api.resetPassword as jest.Mock;
    mockResetPassword.mockResolvedValue({
      data: { message: "パスワードがリセットされました" },
    });

    render(<ResetPasswordForm token={mockToken} />);

    await userEvent.type(
      screen.getByLabelText(/新しいパスワード/i),
      "NewPassword123!"
    );
    await userEvent.type(
      screen.getByLabelText(/パスワード確認/i),
      "NewPassword123!"
    );
    await userEvent.click(
      screen.getByRole("button", { name: /パスワードをリセット/i })
    );

    await waitFor(() => {
      expect(mockResetPassword).toHaveBeenCalledWith(
        mockToken,
        "NewPassword123!"
      );
      expect(window.alert).toHaveBeenCalledWith(
        "パスワードがリセットされました。新しいパスワードでログインしてください。"
      );
      expect(window.location.href).toBe("/login");
    });
  });

  it("パスワードが一致しない場合、バリデーションエラーが表示されること", async () => {
    render(<ResetPasswordForm token={mockToken} />);

    await userEvent.type(
      screen.getByLabelText(/新しいパスワード/i),
      "Password123!"
    );
    await userEvent.type(
      screen.getByLabelText(/パスワード確認/i),
      "DifferentPassword123!"
    );
    await userEvent.click(
      screen.getByRole("button", { name: /パスワードをリセット/i })
    );

    await waitFor(() => {
      expect(screen.getByText("パスワードが一致しません")).toBeInTheDocument();
    });
    expect(api.resetPassword).not.toHaveBeenCalled();
  });

  it("APIエラーが発生した場合、エラーメッセージを表示すること", async () => {
    const mockResetPassword = api.resetPassword as jest.Mock;
    mockResetPassword.mockResolvedValue({ error: "トークンが無効です" });

    render(<ResetPasswordForm token={mockToken} />);

    await userEvent.type(
      screen.getByLabelText(/新しいパスワード/i),
      "NewPassword123!"
    );
    await userEvent.type(
      screen.getByLabelText(/パスワード確認/i),
      "NewPassword123!"
    );
    await userEvent.click(
      screen.getByRole("button", { name: /パスワードをリセット/i })
    );

    await waitFor(() => {
      expect(screen.getByText("トークンが無効です")).toBeInTheDocument();
    });
  });
});
