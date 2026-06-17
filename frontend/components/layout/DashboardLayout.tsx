"use client";
import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { Sidebar } from "@/components/layout/Sidebar";
import { Topbar } from "@/components/layout/Topbar";
import { useAuthStore } from "@/lib/store";

interface DashboardLayoutProps {
  children: React.ReactNode;
  title?: string;
}

const pageVariants = {
  initial: { opacity: 0, y: 8 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.18, ease: "easeOut" } },
};

export default function DashboardLayout({ children, title }: DashboardLayoutProps) {
  const { isAuthenticated } = useAuthStore();
  const router = useRouter();

  // Prevent “UI disappears” caused by the first render while Zustand persisted state hydrates.
  // Zustand persist hydration happens after first client render; during that moment isAuthenticated can be false.
  const [hasHydrated, setHasHydrated] = useState(false);

  useEffect(() => {
    setHasHydrated(true);
  }, []);

  useEffect(() => {
    if (!hasHydrated) return;
    if (!isAuthenticated) router.push("/login");
  }, [hasHydrated, isAuthenticated, router]);

  if (!hasHydrated) return null;

  return (

    <div className="flex h-screen overflow-hidden bg-neutral-50">
      <Sidebar />
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        <Topbar title={title} />
        <main className="flex-1 overflow-y-auto">
          <motion.div
            variants={pageVariants}
            initial="initial"
            animate="animate"
            className="p-6 max-w-7xl mx-auto"
          >
            {children}
          </motion.div>
        </main>
      </div>
    </div>
  );
}
