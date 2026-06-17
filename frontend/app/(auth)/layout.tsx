import type { Metadata } from "next";

export const metadata: Metadata = { title: "Sign In" };

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-neutral-50 via-primary-50 to-neutral-100 flex items-center justify-center p-4">
      {children}
    </div>
  );
}
