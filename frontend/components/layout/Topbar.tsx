"use client";
import React from "react";
import { Bell, Search } from "lucide-react";
import { useAuthStore } from "@/lib/store";
import { getInitials, roleLabels } from "@/lib/utils";

export function Topbar({ title }: { title?: string }) {
  const { user } = useAuthStore();

  return (
    <header className="h-16 bg-white border-b border-neutral-100 flex items-center justify-between px-6 flex-shrink-0">
      <div className="flex items-center gap-4">
        {/* Left spacer for mobile hamburger */}
        <div className="w-8 lg:hidden" />
        {title && (
          <h1 className="text-base font-semibold text-neutral-800 hidden sm:block">{title}</h1>
        )}
      </div>

      <div className="flex items-center gap-3">
        {/* Search (placeholder) */}
        <button className="hidden md:flex items-center gap-2 px-3 py-1.5 rounded-md border border-neutral-200 text-sm text-neutral-500 hover:bg-neutral-50 transition-colors">
          <Search className="h-3.5 w-3.5" />
          <span>Search...</span>
          <kbd className="text-xs bg-neutral-100 px-1.5 py-0.5 rounded border border-neutral-200">⌘K</kbd>
        </button>

        {/* Notifications */}
        <button className="relative p-2 rounded-md text-neutral-500 hover:bg-neutral-100 transition-colors">
          <Bell className="h-5 w-5" />
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-error rounded-full" />
        </button>

        {/* User avatar */}
        {user && (
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center">
              <span className="text-xs font-semibold text-primary-700">
                {getInitials(user.full_name)}
              </span>
            </div>
            <div className="hidden sm:block text-right">
              <p className="text-sm font-medium text-neutral-800 leading-tight">{user.full_name}</p>
              <p className="text-xs text-neutral-500">{roleLabels[user.role] ?? user.role}</p>
            </div>
          </div>
        )}
      </div>
    </header>
  );
}
