import React from "react";
import {
  LayoutDashboard, Users, GraduationCap, BookOpen, DollarSign,
  MessageSquare, Settings, FileText, Bell, Shield, Home, Stethoscope, Bus,
  Library, Package, Building2, UserCheck, Receipt, Calendar, Clock,
} from "lucide-react";
import type { UserProfile } from "./store";

export interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
  permission?: string;
  roles?: string[];
  curriculum?: "844" | "cbc"; // omit = shown for both
}

export const allNavItems: NavItem[] = [
  { label: "Dashboard",      href: "/principal",       icon: LayoutDashboard },
  { label: "Students",       href: "/students",        icon: Users,          permission: "students:read" },
  { label: "Admissions",     href: "/admissions",      icon: UserCheck,      permission: "students:write" },
  { label: "Academics",      href: "/academics",       icon: BookOpen,       permission: "attendance:read" },
  { label: "Classes",        href: "/academics/classes", icon: BookOpen,       permission: "classes:write" },
  { label: "Timetable",      href: "/academics/timetable", icon: Calendar,     permission: "classes:write" },
  { label: "Bell Schedule",  href: "/academics/schedule", icon: Clock,        permission: "classes:write" },
  { label: "Exam Timetable", href: "/academics/exams", icon: Calendar,     permission: "attendance:read" },
  { label: "Report Cards",   href: "/academics/report-cards", icon: FileText, permission: "grades:write", curriculum: "844" },
  { label: "Report Cards",   href: "/academics/cbc-report-cards", icon: FileText, permission: "grades:write", curriculum: "cbc" },
  { label: "Finance",        href: "/finance",         icon: DollarSign,     permission: "fees:read" },
  // fees:manage (migration 012, granted to super_admin) — separate from
  // fees:read since these create/generate billing data rather than just
  // viewing it. Sub-routes of /finance, but given their own entries (not
  // nested under the Finance item) since Sidebar's nav is flat.
  { label: "Fee Structures", href: "/finance/fee-structures", icon: Receipt, permission: "fees:manage" },
  { label: "Generate Invoices", href: "/finance/invoices", icon: FileText, permission: "fees:manage" },
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

export const parentNavItems: NavItem[] = [
  { label: "Dashboard",     href: "/parent",               icon: Home },
  { label: "Fees",          href: "/parent/fees",          icon: DollarSign },
  { label: "Messages",      href: "/parent/messages",      icon: MessageSquare },
  { label: "Announcements", href: "/parent/announcements", icon: Bell },
];

export const teacherNavItems: NavItem[] = [
  { label: "Dashboard",     href: "/teacher",                       icon: LayoutDashboard },
  { label: "Attendance",    href: "/teacher/attendance",            icon: UserCheck },
  { label: "Grades",        href: "/teacher/grades",                icon: BookOpen, curriculum: "844" },
  { label: "Grades",        href: "/teacher/cbc-grades",            icon: BookOpen, curriculum: "cbc" },
  { label: "Report Cards",  href: "/academics/report-cards",        icon: FileText, permission: "grades:write", curriculum: "844" },
  { label: "Report Cards",  href: "/academics/cbc-report-cards",    icon: FileText, permission: "grades:write", curriculum: "cbc" },
  { label: "Assignments",   href: "/teacher/assignments",           icon: FileText },
  { label: "Messages",      href: "/teacher/messages",              icon: MessageSquare },
];

export const studentNavItems: NavItem[] = [
  { label: "Dashboard",  href: "/student",            icon: Home },
  { label: "Timetable",  href: "/student/timetable",  icon: BookOpen },
  { label: "Results",    href: "/student/results",    icon: GraduationCap },
  { label: "Library",    href: "/student/library",    icon: Library },
  { label: "Fees",       href: "/student/fees",       icon: DollarSign },
];

export function getNavItems(role: string): NavItem[] {
  if (role === "parent") return parentNavItems;
  if (role === "teacher") return teacherNavItems;
  if (role === "student") return studentNavItems;
  return allNavItems;
}

// Where a role lands if they hit a route they're not allowed on.
const ROLE_HOME: Record<string, string> = {
  parent: "/parent",
  teacher: "/teacher",
  student: "/student",
};
export function roleHome(role: string): string {
  return ROLE_HOME[role] ?? "/principal";
}

function userHasPermission(user: UserProfile, perm: string): boolean {
  return user.permissions.includes(perm) || user.permissions.includes("*");
}

// Same filter Sidebar uses to decide what to *show* — reused here to decide
// what to *allow*, so the two can never drift apart into two different lists.
function filterVisible(items: NavItem[], user: UserProfile, curriculumType?: string): NavItem[] {
  return items.filter((item) => {
    if (item.curriculum && curriculumType && item.curriculum !== curriculumType) return false;
    if (item.roles) return item.roles.includes(user.role);
    if (item.permission) return userHasPermission(user, item.permission);
    return true;
  });
}

export function isPathAllowed(pathname: string, user: UserProfile, curriculumType?: string): boolean {
  const items = filterVisible(getNavItems(user.role), user, curriculumType);
  return items.some(
    (item) => pathname === item.href || pathname.startsWith(item.href + "/")
  );
}