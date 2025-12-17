"use client"; // クライアントコンポーネントとしてマーク

import AdminLoginForm from "@/features/auth/components/AdminLoginForm/AdminLoginForm";
import Link from "next/link"; // Linkコンポーネントをインポート

export default function AdminLoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-100 p-4">
      <div className="w-full max-w-md space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            管理者アカウントにログイン
          </h2>
        </div>
        <AdminLoginForm />
        <p className="mt-2 text-center text-sm text-gray-600">
          一般ユーザーですか？{" "}
          <Link
            href="/login"
            className="font-medium text-indigo-600 hover:text-indigo-500">
            一般ログインページはこちら
          </Link>
        </p>
      </div>
    </div>
  );
}
