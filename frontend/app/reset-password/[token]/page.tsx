import ResetPasswordForm from "@/features/auth/components/ResetPasswordForm/ResetPasswordForm";

interface ResetPasswordPageProps {
  params: Promise<{ token: string }>;
}

export default async function ResetPasswordPage({
  params,
}: ResetPasswordPageProps) {
  const { token } = await params;
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <ResetPasswordForm token={token} />
    </div>
  );
}
