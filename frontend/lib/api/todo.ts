import { z } from "zod";
import { todoSchema } from "@/app/types/todo";

type Todo = z.infer<typeof todoSchema>;

// クライアントサイドでは常にlocalhostを使用（ブラウザはDocker内部ネットワークにアクセスできない）
const API_BASE_URL =
  typeof window !== "undefined"
    ? "http://localhost:8080"
    : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const todosResponseSchema = z.array(todoSchema);
const todoResponseSchema = todoSchema;
const errorResponseSchema = z.object({
  error: z.string(),
});

export async function fetchTodos(
  token: string
): Promise<z.infer<typeof todosResponseSchema>> {
  try {
    const res = await fetch(`${API_BASE_URL}/api/todos`, {
      cache: "no-store",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    if (!res.ok) {
      let errorMessage = `Failed to fetch todos: ${res.status} ${res.statusText}`;
      try {
        const errorData = await res.json();
        const errorParsed = errorResponseSchema.safeParse(errorData);
        if (errorParsed.success) {
          errorMessage = errorParsed.data.error;
        }
      } catch {
        // JSONパースに失敗した場合はデフォルトメッセージを使用
      }
      throw new Error(errorMessage);
    }

    const data = await res.json();
    const parsed = todosResponseSchema.safeParse(data);
    if (!parsed.success) {
      console.error("Response validation failed:", parsed.error);
      throw new Error("レスポンス形式が無効です");
    }
    return parsed.data;
  } catch (error) {
    // ネットワークエラーまたはCORSエラーの場合
    if (
      error instanceof TypeError &&
      (error.message.includes("fetch") ||
        error.message.includes("Failed to fetch") ||
        error.message.includes("NetworkError"))
    ) {
      throw new Error(
        "バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。"
      );
    }
    throw error;
  }
}

export async function createTodo(
  todo: Omit<Todo, "id" | "user_id" | "created_at" | "updated_at">,
  token: string
): Promise<Todo> {
  try {
    const res = await fetch(`${API_BASE_URL}/api/todos`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(todo),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorParsed = errorResponseSchema.safeParse(errorData);
      throw new Error(
        errorParsed.success
          ? errorParsed.data.error
          : `Failed to create todo: ${res.status} ${res.statusText}`
      );
    }

    const data = await res.json();
    const parsed = todoResponseSchema.safeParse(data);
    if (!parsed.success) {
      console.error("Response validation failed:", parsed.error);
      throw new Error("レスポンス形式が無効です");
    }
    return parsed.data;
  } catch (error) {
    // ネットワークエラーまたはCORSエラーの場合
    if (
      error instanceof TypeError &&
      (error.message.includes("fetch") ||
        error.message.includes("Failed to fetch") ||
        error.message.includes("NetworkError"))
    ) {
      throw new Error(
        "バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。"
      );
    }
    throw error;
  }
}

export async function updateTodo(
  id: number,
  todo: Omit<Todo, "id" | "created_at" | "updated_at">,
  token: string
): Promise<Todo> {
  try {
    const res = await fetch(`${API_BASE_URL}/api/todos/${id}`, {
      method: "PUT",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(todo),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorParsed = errorResponseSchema.safeParse(errorData);
      throw new Error(
        errorParsed.success
          ? errorParsed.data.error
          : `Failed to update todo: ${res.status} ${res.statusText}`
      );
    }

    const data = await res.json();
    const parsed = todoResponseSchema.safeParse(data);
    if (!parsed.success) {
      console.error("Response validation failed:", parsed.error);
      throw new Error("レスポンス形式が無効です");
    }
    return parsed.data;
  } catch (error) {
    // ネットワークエラーまたはCORSエラーの場合
    if (
      error instanceof TypeError &&
      (error.message.includes("fetch") ||
        error.message.includes("Failed to fetch") ||
        error.message.includes("NetworkError"))
    ) {
      throw new Error(
        "バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。"
      );
    }
    throw error;
  }
}

export async function deleteTodo(id: number, token: string): Promise<void> {
  try {
    const res = await fetch(`${API_BASE_URL}/api/todos/${id}`, {
      method: "DELETE",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorParsed = errorResponseSchema.safeParse(errorData);
      throw new Error(
        errorParsed.success
          ? errorParsed.data.error
          : `Failed to delete todo: ${res.status} ${res.statusText}`
      );
    }
  } catch (error) {
    // ネットワークエラーまたはCORSエラーの場合
    if (
      error instanceof TypeError &&
      (error.message.includes("fetch") ||
        error.message.includes("Failed to fetch") ||
        error.message.includes("NetworkError"))
    ) {
      throw new Error(
        "バックエンドサーバーに接続できません。サーバーが起動しているか確認してください。"
      );
    }
    throw error;
  }
}
