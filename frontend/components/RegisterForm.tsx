"use client";

import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { RegisterFormInputs, registerSchema } from "@/app/types/user";
import { registerUser } from "@/lib/api";
import { FaCheckCircle, FaTimesCircle } from "react-icons/fa"; // アイコンをインポート

const RegisterForm: React.FC = () => {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting, touchedFields }, // touchedFields を追加
    setError,
    reset,
  } = useForm<RegisterFormInputs>({
    resolver: zodResolver(registerSchema),
    mode: "onChange",
  });

  const onSubmit = async (data: RegisterFormInputs) => {
    try {
      const result = await registerUser(data);
      if (result.error) {
        setError("root.serverError", { type: "manual", message: result.error });
        console.error("登録エラー:", result.error);
      } else {
        alert("ユーザー登録が成功しました！");
        reset(); // フォームをリセット
      }
    } catch (error) {
      console.error("予期せぬエラー:", error);
      setError("root.serverError", {
        type: "manual",
        message: "予期せぬエラーが発生しました",
      });
    }
  };

  // 各フィールドの状態を判断するためのヘルパー関数
  const getFieldStateClass = (fieldName: keyof RegisterFormInputs) => {
    if (touchedFields[fieldName]) {
      if (errors[fieldName]) {
        return "border-red-500 focus:ring-red-500 focus:border-red-500";
      }
      return "border-green-500 focus:ring-green-500 focus:border-green-500";
    }
    return "border-gray-300 focus:ring-indigo-500 focus:border-indigo-500";
  };

  const getIcon = (fieldName: keyof RegisterFormInputs) => {
    if (!touchedFields[fieldName]) return null;
    if (errors[fieldName]) {
      return <FaTimesCircle className="text-red-500" />;
    }
    return <FaCheckCircle className="text-green-500" />;
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="space-y-4 max-w-md mx-auto p-4 border rounded shadow-lg">
      <div className="relative">
        {" "}
        {/* relative を追加 */}
        <label
          htmlFor="username"
          className="block text-sm font-medium text-gray-700">
          ユーザー名
        </label>
        <input
          id="username"
          type="text"
          {...register("username")}
          className={`mt-1 block w-full px-3 py-2 border rounded-md shadow-sm outline-none ${getFieldStateClass(
            "username"
          )}`}
        />
        <div className="absolute inset-y-0 right-0 pr-3 flex items-center pt-6">
          {" "}
          {/* アイコンのコンテナ */}
          {getIcon("username")}
        </div>
        <div className="min-h-6">
          {" "}
          {/* エラーメッセージの高さ固定 */}
          {errors.username && (
            <p className="mt-1 text-sm text-red-600">
              {errors.username.message}
            </p>
          )}
        </div>
      </div>

      <div className="relative">
        {" "}
        {/* relative を追加 */}
        <label
          htmlFor="email"
          className="block text-sm font-medium text-gray-700">
          メールアドレス
        </label>
        <input
          id="email"
          type="email"
          {...register("email")}
          className={`mt-1 block w-full px-3 py-2 border rounded-md shadow-sm outline-none ${getFieldStateClass(
            "email"
          )}`}
        />
        <div className="absolute inset-y-0 right-0 pr-3 flex items-center pt-6">
          {" "}
          {/* アイコンのコンテナ */}
          {getIcon("email")}
        </div>
        <div className="min-h-6">
          {" "}
          {/* エラーメッセージの高さ固定 */}
          {errors.email && (
            <p className="mt-1 text-sm text-red-600">{errors.email.message}</p>
          )}
        </div>
      </div>

      <div className="relative">
        {" "}
        {/* relative を追加 */}
        <label
          htmlFor="password"
          className="block text-sm font-medium text-gray-700">
          パスワード
        </label>
        <input
          id="password"
          type="password"
          {...register("password")}
          className={`mt-1 block w-full px-3 py-2 border rounded-md shadow-sm outline-none ${getFieldStateClass(
            "password"
          )}`}
        />
        <div className="absolute inset-y-0 right-0 pr-3 flex items-center pt-6">
          {" "}
          {/* アイコンのコンテナ */}
          {getIcon("password")}
        </div>
        <div className="min-h-6">
          {" "}
          {/* エラーメッセージの高さ固定 */}
          {errors.password && (
            <p className="mt-1 text-sm text-red-600">
              {errors.password.message}
            </p>
          )}
        </div>
      </div>

      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50">
        {isSubmitting ? "登録中..." : "登録"}
      </button>
      {/* 修正: root.serverError のメッセージのみを表示 */}
      {errors.root?.serverError && (
        <p className="mt-2 text-sm text-red-600 text-center">
          {errors.root.serverError.message}
        </p>
      )}
    </form>
  );
};
export default RegisterForm;
