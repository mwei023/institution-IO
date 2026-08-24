"use client";

import React, { useState, useEffect} from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { LogOut, ChevronLeft, ChevronRight, Menu, X, GraduationCap } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/lib/store";
import { authAPI } from "@/lib/api";
import { useRouter } from "next/navigation";
import { useInstitutionStore } from "@/lib/institutionStore";
import { getNavItems, allNavItems } from "@/lib/access";


export function Sidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const pathname = usePathname();
  const { user, hasPermission, hasRole, clearAuth } = useAuthStore();
  const { context, fetchContext, clearContext } = useInstitutionStore();
  const router = useRouter();

  useEffect(() => {
    if (user) fetchContext();
  }, [user, fetchContext]);

  const navItems = user ? getNavItems(user.role) : allNavItems;

  const visibleItems = navItems.filter((item) => {
    if (item.curriculum && context && item.curriculum !== context.curriculum_type) return false;
    if (item.roles && user) return item.roles.includes(user.role);
    if (item.permission) return hasPermission(item.permission);
    return true;
  });

  const handleLogout = async () => {
    try { await authAPI.logout(); } catch (_) {}
    clearAuth();
    clearContext();
    router.push("/login");
  };

  const SidebarContent = () => (
    <div className="flex flex-col h-full">
      {/* Logo */}
      <div className={cn(
        "flex items-center gap-3 px-4 py-5 border-b border-neutral-100",
        collapsed && "justify-center px-2"
      )}>
        <div className="flex-shrink-0 w-8 h-8 rounded-md bg-primary-600 flex items-center justify-center">
          <GraduationCap className="h-5 w-5 text-white" />
        </div>
        {!collapsed && (
          <div>
            <p className="text-sm font-semibold text-neutral-900 leading-tight">Institution OS</p>
            <p className="text-xs text-neutral-400 truncate max-w-[140px]">
              {user?.full_name ?? "Loading..."}
            </p>
          </div>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-4 px-2 space-y-0.5">
        {visibleItems.map((item) => {
          const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={() => setMobileOpen(false)}
              className={cn(
                "nav-item",
                isActive && "nav-item-active",
                collapsed && "justify-center px-2"
              )}
              title={collapsed ? item.label : undefined}
            >
              <Icon className="h-4 w-4 flex-shrink-0" />
              {!collapsed && <span>{item.label}</span>}
            </Link>
          );
        })}
      </nav>

      {/* Footer: collapse toggle + logout */}
      <div className="border-t border-neutral-100 p-2 space-y-0.5">
        <button
          onClick={handleLogout}
          className={cn(
            "nav-item w-full text-error hover:bg-error-light hover:text-error",
            collapsed && "justify-center px-2"
          )}
        >
          <LogOut className="h-4 w-4 flex-shrink-0" />
          {!collapsed && <span>Sign Out</span>}
        </button>

        {/* Desktop collapse toggle */}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="hidden lg:flex nav-item w-full justify-center px-2"
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          {collapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <ChevronLeft className="h-4 w-4" />
          )}
        </button>
      </div>
    </div>
  );

  return (
    <>
      {/* Mobile hamburger */}
      <button
        onClick={() => setMobileOpen(true)}
        className="fixed top-4 left-4 z-50 lg:hidden p-2 rounded-md bg-white shadow border border-neutral-200"
      >
        <Menu className="h-5 w-5 text-neutral-600" />
      </button>

      {/* Mobile overlay */}
      <AnimatePresence>
        {mobileOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="fixed inset-0 z-40 bg-neutral-900/50 lg:hidden"
            onClick={() => setMobileOpen(false)}
          />
        )}
      </AnimatePresence>

      {/* Mobile drawer */}
      <AnimatePresence>
        {mobileOpen && (
          <motion.div
            initial={{ x: -280 }}
            animate={{ x: 0 }}
            exit={{ x: -280 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className="fixed inset-y-0 left-0 z-50 w-72 bg-white shadow-modal lg:hidden"
          >
            <button
              onClick={() => setMobileOpen(false)}
              className="absolute top-4 right-4 p-1.5 rounded-md text-neutral-500 hover:bg-neutral-100"
            >
              <X className="h-4 w-4" />
            </button>
            <SidebarContent />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Desktop sidebar */}
      <motion.div
        animate={{ width: collapsed ? 64 : 240 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="hidden lg:flex flex-col h-screen bg-white border-r border-neutral-100 flex-shrink-0 sticky top-0 overflow-hidden"
      >
        <SidebarContent />
      </motion.div>
    </>
  );
}