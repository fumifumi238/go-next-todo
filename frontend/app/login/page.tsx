"use client"; // クライアントコンポーネントとしてマーク

import LoginForm from "@/features/auth/components/LoginForm/LoginForm";
import Link from "next/link"; // Linkコンポーネントをインポート

export default function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-100 p-4">
      <div className="w-full max-w-md space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            アカウントにログイン
          </h2>
        </div>
        <LoginForm />
        <p className="mt-2 text-center text-sm text-gray-600">
          アカウントをお持ちでないですか？{" "}
          <Link
            href="/register"
            className="font-medium text-indigo-600 hover:text-indigo-500">
            新規登録はこちら
          </Link>
        </p>
      </div>
    </div>
  );
}
