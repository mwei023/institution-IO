"use client";

import React, { useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import { Loader2 } from "lucide-react";
import { useAuthStore } from "@/lib/store";
import { useInstitutionStore } from "@/lib/institutionStore";
import { isPathAllowed, roleHome } from "@/lib/access";

export default function DashboardGroupLayout({ children }: { children: React.ReactNode }) {
  const { user, hasHydrated } = useAuthStore();
  const { context } = useInstitutionStore();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    // Persisted auth state loads asynchronously from localStorage — deciding
    // anything before it's ready would wrongly bounce a logged-in person to
    // /login on every hard refresh. Wait for it explicitly.
    if (!hasHydrated) return;

    if (!user) {
      router.replace("/login");
      return;
    }

    if (!isPathAllowed(pathname, user, context?.curriculum_type)) {
      router.replace(roleHome(user.role));
    }
  }, [hasHydrated, user, pathname, context, router]);

  if (!hasHydrated) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
      </div>
    );
  }

  // Redirecting — render nothing so the disallowed page never flashes on screen.
  if (!user) return null;
  if (!isPathAllowed(pathname, user, context?.curriculum_type)) return null;

  return <>{children}</>;
}