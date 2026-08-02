"use client";

import React, { useState, useEffect} from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import {
  LayoutDashboard, Users, GraduationCap, BookOpen, DollarSign,
  MessageSquare, Settings, LogOut, ChevronLeft, ChevronRight,
  FileText, Bell, Shield, Home, Stethoscope, Bus, Library,
  Package, Building2, UserCheck, Menu, X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/lib/store";
import { authAPI } from "@/lib/api";
import { useRouter } from "next/navigation";
import { useInstitutionStore } from "@/lib/institutionStore";

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
  permission?: string;
  roles?: string[];
  curriculum?: "844" | "cbc"; // omit = shown for both
}

const allNavItems: NavItem[] = [
  { label: "Dashboard",      href: "/principal",       icon: LayoutDashboard },
  { label: "Students",       href: "/students",        icon: Users,          permission: "students:read" },
  { label: "Admissions",     href: "/admissions",      icon: UserCheck,      permission: "students:write" },
  { label: "Academics",      href: "/academics",       icon: BookOpen,       permission: "attendance:read" },
  { label: "Report Cards",   href: "/academics/report-cards", icon: FileText, permission: "grades:write", curriculum: "844" },
  { label: "Finance",        href: "/finance",         icon: DollarSign,     permission: "fees:read" },
  { label: "Staff",          href: "/staff",           icon: GraduationCap,  permission: "staff:read" },
  { label: "Communications", href: "/communications",  icon: MessageSquare,  permission: "announcements:read" },
  { label: "Documents",      href: "/documents",       icon: FileText,       permission: "documents:read" },
  { label: "Clinic",         href: "/clinic",          icon: Stethoscope,    roles: ["nurse", "super_admin"] },
  { label: "Library",        href: "/library",         icon: Library,        roles: ["librarian", "super_admin"] },
  { label: "Transport",      href: "/transport",       icon: Bus,            roles: ["driver", "super_admin"] },
  { label: "Inventory",      href: "/inventory",       icon: Package,        roles: ["storekeeper", "super_admin"] },
  { label: "Boarding",       href: "/boarding",        icon: Building2,      roles: ["super_admin"] },
  { label: "Security",       href: "/security",        icon: Shield,         roles: ["security_officer", "super_admin"] },
  { label: "Settings",       href: "/settings",        icon: Settings,       permission: "settings:read" },
];

const parentNavItems: NavItem[] = [
  { label: "Dashboard",    href: "/parent",   icon: Home },
  { label: "Fees",         href: "/parent/fees",     icon: DollarSign },
  { label: "Messages",     href: "/parent/messages", icon: MessageSquare },
  { label: "Announcements",href: "/parent/announcements", icon: Bell },
];

const teacherNavItems: NavItem[] = [
  { label: "Dashboard",    href: "/teacher",            icon: LayoutDashboard },
  { label: "Attendance",   href: "/teacher/attendance", icon: UserCheck },
  { label: "Grades",       href: "/teacher/grades",     icon: BookOpen, curriculum: "844" },
  { label: "Report Cards",   href: "/academics/report-cards", icon: FileText, permission: "grades:write", curriculum: "844" },
  { label: "Report Cards",   href: "/academics/cbc-report-cards", icon: FileText, permission: "grades:write", curriculum: "cbc" },
  { label: "Assignments",  href: "/teacher/assignments",icon: FileText },
  { label: "Messages",     href: "/teacher/messages",   icon: MessageSquare },
];

const studentNavItems: NavItem[] = [
  { label: "Dashboard",   href: "/student",            icon: Home },
  { label: "Timetable",   href: "/student/timetable",  icon: BookOpen },
  { label: "Results",     href: "/student/results",    icon: GraduationCap },
  { label: "Library",     href: "/student/library",    icon: Library },
  { label: "Fees",        href: "/student/fees",       icon: DollarSign },
];

function getNavItems(role: string) {
  if (role === "parent") return parentNavItems;
  if (role === "teacher") return teacherNavItems;
  if (role === "student") return studentNavItems;
  return allNavItems;
}

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