import { z } from "zod";

export const todoSchema = z.object({
  id: z.number().optional(),
  user_id: z.number().optional(),
  title: z.string().min(1, { message: "タイトルは必須です" }),
  completed: z.boolean(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});

export type Todo = z.infer<typeof todoSchema>;
