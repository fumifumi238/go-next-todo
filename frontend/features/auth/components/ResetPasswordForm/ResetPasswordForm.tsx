"use client";

import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ResetPasswordFormInputs, resetPasswordSchema } from "@/app/types/user";
import { resetPassword as resetPasswordAPI } from "@/lib/api";
import FieldStatus from "@/features/form/FieldStatus";

interface ResetPasswordFormProps {
  token: string;
}

const ResetPasswordForm: React.FC<ResetPasswordFormProps> = ({ token }) => {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
    reset,
    watch,
  } = useForm<ResetPasswordFormInputs>({
    resolver: zodResolver(resetPasswordSchema),
  });

  const password = watch("password");

  const onSubmit = async (data: ResetPasswordFormInputs) => {
    if (!token || token === "undefined" || token.trim() === "") {
      setError("root.serverError", {
        type: "manual",
        message:
          "無効なトークンです。パスワードリセットリンクを確認してください。",
      });
      return;
    }

    try {
      const result = await resetPasswordAPI(token, data.password);
      if (result.error) {
        setError("root.serverError", {
          type: "manual",
          message: result.error,
        });
      } else {
        alert(
          "パスワードがリセットされました。新しいパスワードでログインしてください。"
        );
        reset();
        // ログインページにリダイレクト
        window.location.href = "/login";
      }
    } catch (error) {
      console.error("予期せぬエラー:", error);
      setError("root.serverError", {
        type: "manual",
        message: "予期せぬエラーが発生しました",
      });
    }
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="space-y-2 max-w-md mx-auto p-4 border rounded shadow-lg">
      <h2 className="text-xl font-bold text-center">パスワードをリセット</h2>
      <div className="space-y-2">
        <div className="relative">
          <label
            htmlFor="password"
            className="block text-sm font-medium text-gray-700">
            新しいパスワード
          </label>
          <div className="mt-1 relative rounded-md shadow-sm">
            <input
              id="password"
              type="text"
              {...register("password")}
              className="block w-full pl-10 pr-3 py-2 border rounded-md outline-none border-gray-300 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="8文字以上、大文字小文字数字記号"
            />
          </div>
          <FieldStatus value={password} error={errors.password?.message} />
        </div>
        <div className="relative">
          <label
            htmlFor="confirmPassword"
            className="block text-sm font-medium text-gray-700">
            パスワード確認
          </label>
          <div className="mt-1 relative rounded-md shadow-sm">
            <input
              id="confirmPassword"
              type="text"
              {...register("confirmPassword")}
              className="block w-full pl-10 pr-3 py-2 border rounded-md outline-none border-gray-300 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="パスワードを再度入力"
            />
          </div>
          <FieldStatus
            value={watch("confirmPassword")}
            error={errors.confirmPassword?.message}
          />
        </div>
      </div>
      <button
        type="submit"
        disabled={isSubmitting || !password || !watch("confirmPassword")}
        className="w-full flex justify-center py-2 px-4 rounded-md shadow-sm text-sm font-medium bg-primary text-primary-foreground border border-transparent transition-opacity duration-200 hover:opacity-90 disabled:opacity-60 disabled:cursor-not-allowed disabled:hover:opacity-60">
        {isSubmitting ? "リセット中..." : "パスワードをリセット"}
      </button>
      {errors.root?.serverError && (
        <p className="mt-2 text-sm text-red-600 text-center">
          {errors.root.serverError.message}
        </p>
      )}
    </form>
  );
};

export default ResetPasswordForm;
