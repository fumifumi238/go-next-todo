"use client";

import React, { useContext } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { RegisterFormInputs, registerSchema } from "@/app/types/user";
import { registerUser, loginUser } from "@/lib/api";
import { useRouter } from "next/navigation";
import FieldStatus from "@/components/FieldStatus";
import { AuthContext } from "@/context/AuthContext";

const RegisterForm: React.FC = () => {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting, touchedFields, isValid }, // touchedFields を追加
    setError,
    reset,
    control,
  } = useForm<RegisterFormInputs>({
    resolver: zodResolver(registerSchema),
    mode: "onChange",
  });

  const { login } = useContext(AuthContext);
  const router = useRouter();

  const username = useWatch({ control, name: "username" });
  const email = useWatch({ control, name: "email" });
  const password = useWatch({ control, name: "password" });

  // 空白チェック
  const isAllFilled = username && email && password;

  const onSubmit = async (data: RegisterFormInputs) => {
    try {
      const result = await registerUser(data);
      if (result.error) {
        setError("root.serverError", {
          type: "manual",
          message: result.error,
        });
        console.error("登録エラー:", result.error);
      } else {
        alert("ユーザー登録が成功しました！自動的にログインします。");
        // 登録成功後、直ちにログイン処理を行う
        const loginResponse = await loginUser({
          email: data.email,
          password: data.password,
        });

        if (loginResponse.error) {
          setError("root.serverError", {
            type: "manual",
            message: `自動ログインに失敗しました: ${loginResponse.error}`,
          });
          console.error("自動ログインエラー:", loginResponse.error);
          router.push("/login"); // 自動ログイン失敗時はログインページへ
        } else if (loginResponse.data?.token) {
          login(loginResponse.data.token); // ログイン成功時、AuthContextにトークンをセット
          reset(); // フォームをリセット
          router.push("/"); // ログイン成功後にルートパスにリダイレクト
        } else {
          setError("root.serverError", {
            type: "manual",
            message:
              "自動ログイン時に予期せぬAPIレスポンス: トークンが見つかりません。",
          });
          console.error(
            "自動ログイン時に予期せぬAPIレスポンス:",
            loginResponse
          );
          router.push("/login"); // 予期せぬ応答時はログインページへ
        }
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
  const getFieldStateclassName = (fieldName: keyof RegisterFormInputs) => {
    if (touchedFields[fieldName]) {
      if (errors[fieldName]) {
        return "border-red-500 focus:ring-red-500 focus:border-red-500";
      }
      return "border-green-500 focus:ring-green-500 focus:border-green-500";
    }
    return "border-gray-300 focus:ring-indigo-500 focus:border-indigo-500";
  };
  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="space-y-2 max-w-md mx-auto p-4 border rounded shadow-lg">
      <div className="space-y-2">
        <div className="relative">
          <label
            htmlFor="username"
            className="block text-sm font-medium text-gray-700">
            ユーザー名
          </label>
          <div className="mt-1 relative rounded-md shadow-sm">
            <input
              id="username"
              type="text"
              {...register("username")}
              className={`block w-full pl-10 pr-3 py-2 border rounded-md outline-none ${getFieldStateclassName(
                "username"
              )}`} // ここで pl-10 を追加
            />
          </div>
          <FieldStatus value={username} error={errors.username?.message} />
        </div>
        <div className="relative">
          <label
            htmlFor="email"
            className="block text-sm font-medium text-gray-700">
            メールアドレス
          </label>
          <div className="mt-0.5 relative rounded-md shadow-sm">
            <input
              id="email"
              type="email"
              {...register("email")}
              className={`block w-full pl-10 pr-3 py-2 border rounded-md outline-none ${getFieldStateclassName(
                "email"
              )}`} // ここで pl-10 を追加
            />
          </div>
          <FieldStatus value={email} error={errors.email?.message} />
        </div>

        <div className="relative">
          <label
            htmlFor="password"
            className="block text-sm font-medium text-gray-700">
            パスワード
          </label>
          <div className="mt-0.5 relative rounded-md shadow-sm">
            <input
              id="password"
              type="password"
              {...register("password")}
              className={`block w-full pl-10 pr-3 py-2 border rounded-md outline-none ${getFieldStateclassName(
                "password"
              )}`} // ここで pl-10 を追加
            />
          </div>
          <FieldStatus value={password} error={errors.password?.message} />
        </div>
      </div>
      <button
        type="submit"
        disabled={isSubmitting || !isAllFilled || !isValid}
        className={`
    w-full flex justify-center py-2 px-4 rounded-md shadow-sm text-sm font-medium
    bg-primary text-primary-foreground border border-transparent
    transition-opacity duration-200

    hover:opacity-90
    disabled:opacity-60 disabled:cursor-not-allowed disabled:hover:opacity-60
  `}>
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
