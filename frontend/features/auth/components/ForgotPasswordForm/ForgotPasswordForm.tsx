"use client";

import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { forgotPassword } from "@/lib/api";
import FieldStatus from "@/features/form/FieldStatus";

const forgotPasswordSchema = z.object({
  email: z.email({ message: "有効なメールアドレスを入力してください" }),
});

type ForgotPasswordInputs = z.infer<typeof forgotPasswordSchema>;

const ForgotPasswordForm: React.FC = () => {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
    reset,
    watch,
  } = useForm<ForgotPasswordInputs>({
    resolver: zodResolver(forgotPasswordSchema),
    mode: "onChange",
  });

  const email = watch("email");

  const onSubmit = async (data: ForgotPasswordInputs) => {
    try {
      const result = await forgotPassword(data.email);
      if (result.error) {
        setError("root.serverError", {
          type: "manual",
          message: result.error,
        });
      } else {
        alert(
          "パスワードリセットのメールを送信しました。メールを確認してください。"
        );
        reset();
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
      <h2 className="text-xl font-bold text-center">パスワードを忘れた場合</h2>
      <div className="space-y-2">
        <div className="relative">
          <label
            htmlFor="email"
            className="block text-sm font-medium text-gray-700">
            メールアドレス
          </label>
          <div className="mt-1 relative rounded-md shadow-sm">
            <input
              id="email"
              type="email"
              {...register("email")}
              className="block w-full pl-10 pr-3 py-2 border rounded-md outline-none border-gray-300 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <FieldStatus value={email} error={errors.email?.message} />
        </div>
      </div>
      <button
        type="submit"
        disabled={isSubmitting || !email}
        className="w-full flex justify-center py-2 px-4 rounded-md shadow-sm text-sm font-medium bg-primary text-primary-foreground border border-transparent transition-opacity duration-200 hover:opacity-90 disabled:opacity-60 disabled:cursor-not-allowed disabled:hover:opacity-60">
        {isSubmitting ? "送信中..." : "送信"}
      </button>
      {errors.root?.serverError && (
        <p className="mt-2 text-sm text-red-600 text-center">
          {errors.root.serverError.message}
        </p>
      )}
    </form>
  );
};

export default ForgotPasswordForm;
