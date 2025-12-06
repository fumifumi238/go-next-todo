"use client";

import React, { useContext } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { LoginFormInputs, loginSchema } from "@/app/types/user";
import { loginUser } from "@/lib/api"; // ログインAPIをインポート
import FieldStatus from "@/components/FieldStatus";
import { AuthContext } from "@/context/AuthContext"; // AuthContextをインポート
import { useRouter } from "next/navigation";

const LoginForm: React.FC = () => {
  const { login } = useContext(AuthContext); // AuthContextからlogin関数を取得
  const router = useRouter();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting, touchedFields, isValid },
    setError,
    reset,
    control,
  } = useForm<LoginFormInputs>({
    resolver: zodResolver(loginSchema),
    mode: "onChange",
  });

  const email = useWatch({ control, name: "email" });
  const password = useWatch({ control, name: "password" });
  // 空白チェック (usernameは不要)
  const isAllFilled = email && password;

  const onSubmit = async (data: LoginFormInputs) => {
    try {
      const response = await loginUser(data);

      if (response.error) {
        setError("root.serverError", {
          type: "manual",
          message: response.error,
        });
        console.error("ログインエラー:", response.error);
        return; // エラーの場合はここで処理を終了
      }

      // ログイン成功時の処理
      if (response.data?.token) {
        login(response.data.token);
        alert("ログインに成功しました！");
        reset();
        router.push("/");
      } else {
        // 予期せぬAPIレスポンス (dataまたはtokenがない場合)
        const errorMessage =
          "予期せぬAPIレスポンス: トークンが見つかりません。";
        setError("root.serverError", {
          type: "manual",
          message: errorMessage,
        });
        console.error(errorMessage, response);
      }
    } catch (error) {
      // ネットワークエラーなどの予期せぬ例外
      const errorMessage = "ネットワークエラーによりログインに失敗しました。";
      setError("root.serverError", {
        type: "manual",
        message: errorMessage,
      });
      console.error("予期せぬエラー:", error);
    }
  };

  // 各フィールドの状態を判断するためのヘルパー関数
  const getFieldStateClassName = (fieldName: keyof LoginFormInputs) => {
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
            className={`block w-full pl-10 pr-3 py-2 border rounded-md outline-none ${getFieldStateClassName(
              "email"
            )}`}
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
            className={`block w-full pl-10 pr-3 py-2 border rounded-md outline-none ${getFieldStateClassName(
              "password"
            )}`}
          />
        </div>
        <FieldStatus value={password} error={errors.password?.message} />
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
        {isSubmitting ? "ログイン中..." : "ログイン"}
      </button>
      {errors.root?.serverError && (
        <p className="mt-2 text-sm text-red-600 text-center">
          {errors.root.serverError.message}
        </p>
      )}
    </form>
  );
};
export default LoginForm;
