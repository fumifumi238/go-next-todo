"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { createTodo } from "@/lib/api/todo";

// max(255文字)を設定する方が良いです。またタイトルは必須ですとするより、文字列がないときにボタンを押せないようにする方が適切です。
const todoFormSchema = z.object({
  title: z.string().min(1, { message: "タイトルは必須です" }),
  completed: z.boolean(),
});

type TodoFormInputs = z.infer<typeof todoFormSchema>;

type TodoFormProps = {
  onAdd: () => void;
  token: string;
};

export default function TodoForm({ onAdd, token }: TodoFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    reset,
    watch,
  } = useForm<TodoFormInputs>({
    resolver: zodResolver(todoFormSchema),
    defaultValues: {
      title: "",
      completed: false,
    },
  });

  // watchではなくuseWatchを使ってください
  const title = watch("title");

  const onSubmit = async (data: TodoFormInputs) => {
    // todoを作成したときには必ずfalseとなります。completedはfalse、またはdbのデフォルト設定に任せて送信しないのが良いと思います。
    try {
      await createTodo(
        {
          title: data.title.trim(),
          completed: data.completed,
        },
        token
      );
      reset();
      onAdd();
    } catch (error) {
      console.error("Failed to create todo:", error);
      alert("TODOの作成に失敗しました");
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="mb-6">
      <div className="flex gap-2">
        <input
          type="text"
          {...register("title")}
          placeholder="新しいTODOを入力..."
          disabled={isSubmitting}
          className="flex-1 px-4 py-2 border border-border rounded-lg bg-card text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={isSubmitting || !title?.trim()}
          className="px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed">
          {isSubmitting ? "追加中..." : "追加"}
        </button>
      </div>
      {errors.title && (
        <p className="text-red-500 text-sm mt-1">{errors.title.message}</p>
      )}
    </form>
  );
}
