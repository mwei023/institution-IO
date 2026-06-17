import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { format, formatDistanceToNow } from "date-fns";

/** Merge Tailwind classes safely */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Format KES currency */
export function formatKES(amount: number): string {
  return new Intl.NumberFormat("en-KE", {
    style: "currency",
    currency: "KES",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

/** Format a date as "15 Jan 2024" */
export function formatDate(date: string | Date): string {
  return format(new Date(date), "d MMM yyyy");
}

/** Format a date as "15 Jan 2024 at 10:30 AM" */
export function formatDateTime(date: string | Date): string {
  return format(new Date(date), "d MMM yyyy 'at' h:mm a");
}

/** Relative time: "3 days ago" */
export function timeAgo(date: string | Date): string {
  return formatDistanceToNow(new Date(date), { addSuffix: true });
}

/** Compute age from date of birth */
export function computeAge(dob: string | Date): number {
  const birthDate = new Date(dob);
  const today = new Date();
  let age = today.getFullYear() - birthDate.getFullYear();
  const m = today.getMonth() - birthDate.getMonth();
  if (m < 0 || (m === 0 && today.getDate() < birthDate.getDate())) age--;
  return age;
}

/** Get initials from a name */
export function getInitials(name: string): string {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((n) => n[0].toUpperCase())
    .join("");
}

/** Format phone for display: +254712345678 → 0712 345 678 */
export function formatPhone(phone: string): string {
  if (!phone) return "";
  const clean = phone.replace(/\D/g, "");
  if (clean.startsWith("254") && clean.length === 12) {
    const local = "0" + clean.slice(3);
    return `${local.slice(0, 4)} ${local.slice(4, 7)} ${local.slice(7)}`;
  }
  return phone;
}

/** Status badge colour mapping */
export const statusColors: Record<string, string> = {
  active:    "bg-success-light text-success-dark",
  inactive:  "bg-neutral-100 text-neutral-600",
  graduated: "bg-primary-50 text-primary-700",
  enrolled:  "bg-success-light text-success-dark",
  submitted: "bg-warning-light text-warning-dark",
  approved:  "bg-success-light text-success-dark",
  rejected:  "bg-error-light text-error-dark",
  pending:   "bg-warning-light text-warning-dark",
  paid:      "bg-success-light text-success-dark",
  partial:   "bg-warning-light text-warning-dark",
  unpaid:    "bg-error-light text-error-dark",
  waived:    "bg-neutral-100 text-neutral-600",
  draft:     "bg-neutral-100 text-neutral-600",
  published: "bg-success-light text-success-dark",
  present:   "bg-success-light text-success-dark",
  absent:    "bg-error-light text-error-dark",
  late:      "bg-warning-light text-warning-dark",
  excused:   "bg-neutral-100 text-neutral-600",
};

/** Role display names */
export const roleLabels: Record<string, string> = {
  super_admin:      "Principal / Admin",
  dept_admin:       "Department Admin",
  teacher:          "Teacher",
  student:          "Student",
  parent:           "Parent",
  accountant:       "Accountant",
  nurse:            "Nurse",
  librarian:        "Librarian",
  storekeeper:      "Storekeeper",
  driver:           "Driver",
  security_officer: "Security Officer",
  board_member:     "Board Member",
};

/** Role → default dashboard path */
export const roleDashboardPath: Record<string, string> = {
  super_admin:      "/principal",
  dept_admin:       "/principal",
  teacher:          "/teacher",
  student:          "/student",
  parent:           "/parent",
  accountant:       "/principal",
  nurse:            "/principal",
  librarian:        "/principal",
  storekeeper:      "/principal",
  driver:           "/principal",
  security_officer: "/principal",
  board_member:     "/principal",
};
