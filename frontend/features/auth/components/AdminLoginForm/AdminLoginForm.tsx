"use client";

import React, { useContext, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { LoginFormInputs, loginSchema } from "@/app/types/user";
import { adminLoginStep1, adminLoginStep2 } from "@/lib/api/user"; // 管理者ログインAPIをインポート
import FieldStatus from "@/features/form/FieldStatus";
import { AuthContext } from "@/context/AuthContext"; // AuthContextをインポート
import { useRouter } from "next/navigation";

const AdminLoginForm: React.FC = () => {
  const { login } = useContext(AuthContext); // AuthContextからlogin関数を取得
  const router = useRouter();
  const [step, setStep] = useState<"password" | "otp">("password");
  const [userId, setUserId] = useState<number | null>(null);
  const [otp, setOtp] = useState("");
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
      const response = await adminLoginStep1(data);
      setUserId(response.user_id);
      setStep("otp");
    } catch (error) {
      setError("root.serverError", {
        type: "manual",
        message:
          error instanceof Error
            ? error.message
            : "管理者ログインに失敗しました",
      });
    }
  };

  const onOtpSubmit = async () => {
    if (!userId || otp.length !== 6) return;
    try {
      const response = await adminLoginStep2(userId, otp);
      login(response.token, "admin");
      alert("管理者ログインに成功しました！");
      reset();
      setStep("password");
      setUserId(null);
      setOtp("");
      router.push("/");
    } catch (error) {
      setError("root.serverError", {
        type: "manual",
        message:
          error instanceof Error ? error.message : "OTP検証に失敗しました",
      });
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
      onSubmit={
        step === "password"
          ? handleSubmit(onSubmit)
          : (e) => {
              e.preventDefault();
              onOtpSubmit();
            }
      }
      className="space-y-2 max-w-md mx-auto p-4 border rounded shadow-lg">
      {step === "password" && (
        <>
          <div className="relative">
            <label
              htmlFor="email"
              className="block text-sm font-medium text-gray-700">
              管理者メールアドレス
            </label>
            <div className="mt-1 relative rounded-md shadow-sm">
              <input
                id="email"
                type="email"
                placeholder="admin@sample.com"
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
                placeholder="8文字以上＋英大/小＋数字＋記号"
                {...register("password")}
                className={`block w-full pl-10 pr-3 py-2 border rounded-md outline-none ${getFieldStateClassName(
                  "password"
                )}`}
              />
            </div>
            <FieldStatus value={password} error={errors.password?.message} />
          </div>
        </>
      )}

      {step === "otp" && (
        <div className="relative">
          <label
            htmlFor="otp"
            className="block text-sm font-medium text-gray-700">
            OTPコード (6桁)
          </label>
          <div className="mt-1 relative rounded-md shadow-sm">
            <input
              id="otp"
              type="text"
              placeholder="123456"
              value={otp}
              onChange={(e) => setOtp(e.target.value)}
              className="block w-full pl-10 pr-3 py-2 border rounded-md outline-none border-gray-300 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
        </div>
      )}

      <button
        type="submit"
        disabled={
          step === "password"
            ? isSubmitting || !isAllFilled || !isValid
            : otp.length !== 6
        }
        className={`
    w-full flex justify-center py-2 px-4 rounded-md shadow-sm text-sm font-medium
    bg-primary text-primary-foreground border border-transparent
    transition-opacity duration-200

    hover:opacity-90
    disabled:opacity-60 disabled:cursor-not-allowed disabled:hover:opacity-60
  `}>
        {isSubmitting
          ? "処理中..."
          : step === "password"
          ? "ログイン"
          : "OTP検証"}
      </button>
      {errors.root?.serverError && (
        <p className="mt-2 text-sm text-red-600 text-center">
          {errors.root.serverError.message}
        </p>
      )}
      {step === "otp" && (
        <button
          type="button"
          onClick={() => {
            setStep("password");
            setUserId(null);
            setOtp("");
          }}
          className="w-full mt-2 text-sm text-blue-600 hover:text-blue-800">
          戻る
        </button>
      )}
    </form>
  );
};
export default AdminLoginForm;
