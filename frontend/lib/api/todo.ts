import { Todo } from "@/app//types/todo";

// クライアントサイドでは常にlocalhostを使用（ブラウザはDocker内部ネットワークにアクセスできない）
const API_URL =
  typeof window !== "undefined"
    ? "http://localhost:8080"
    : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function fetchTodos(): Promise<Todo[]> {
  try {
    const res = await fetch(`${API_URL}/api/todos`, {
      cache: "no-store",
    });

    if (!res.ok) {
      let errorMessage = `Failed to fetch todos: ${res.status} ${res.statusText}`;
      try {
        const errorData = await res.json();
        if (errorData.error) {
          errorMessage = errorData.error;
        }
        if (errorData.details) {
          errorMessage += ` (${errorData.details})`;
        }
      } catch {
        // JSONパースに失敗した場合はデフォルトメッセージを使用
      }
      throw new Error(errorMessage);
    }

    return res.json();
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
  todo: Omit<Todo, "id" | "created_at">
): Promise<Todo> {
  try {
    const res = await fetch(`${API_URL}/api/todos`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(todo),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      throw new Error(
        errorData.error ||
          `Failed to create todo: ${res.status} ${res.statusText}`
      );
    }

    return res.json();
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
  todo: Omit<Todo, "id" | "created_at">
): Promise<Todo> {
  try {
    const res = await fetch(`${API_URL}/api/todos/${id}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(todo),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      throw new Error(
        errorData.error ||
          `Failed to update todo: ${res.status} ${res.statusText}`
      );
    }

    return res.json();
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

export async function deleteTodo(id: number): Promise<void> {
  try {
    const res = await fetch(`${API_URL}/api/todos/${id}`, {
      method: "DELETE",
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      throw new Error(
        errorData.error ||
          `Failed to delete todo: ${res.status} ${res.statusText}`
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
