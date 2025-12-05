import { z } from "zod";

// ユーザー登録リクエストのZodスキーマ
export const registerSchema = z.object({
  username: z
    .string()
    .min(3, { message: "ユーザー名は3文字以上である必要があります" }),
  email: z.email({ message: "有効なメールアドレスを入力してください" }),
  password: z
    .string()
    .min(8, { message: "パスワードは8文字以上である必要があります" })
    .regex(/[A-Z]/, {
      message: "パスワードには大文字が1文字以上含まれている必要があります",
    })
    .regex(/[a-z]/, {
      message: "パスワードには小文字が1文字以上含まれている必要があります",
    })
    .regex(/[0-9]/, {
      message: "パスワードには数字が1文字以上含まれている必要があります",
    })
    .regex(/[^a-zA-Z0-9]/, {
      message: "パスワードには記号が1文字以上含まれている必要があります",
    }),
});

// ユーザー登録リクエストの型定義
export type RegisterFormInputs = z.infer<typeof registerSchema>;

// ユーザーログインリクエストのZodスキーマ (将来的に使用)
export const loginSchema = z.object({
  email: z
    .email({ message: "有効なメールアドレスを入力してください" })
    .max(255, { message: "メールアドレスは255文字以下で入力してください" }),
  password: z
    .string()
    .min(1, { message: "パスワードを入力してください" }) // ログイン時は最小文字数のバリデーションは不要な場合が多い
    .max(100, { message: "パスワードは100文字以下で入力してください" }),
});
// ユーザーログインリクエストの型定義
export type LoginFormInputs = z.infer<typeof loginSchema>;
