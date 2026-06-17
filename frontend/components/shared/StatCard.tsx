"use client";
import React from "react";
import { cn } from "@/lib/utils";
import { LucideIcon } from "lucide-react";

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  icon?: LucideIcon;
  iconColor?: string;
  trend?: { value: number; label: string };
  className?: string;
  onClick?: () => void;
}

export function StatCard({
  label,
  value,
  sub,
  icon: Icon,
  iconColor = "text-primary-500",
  trend,
  className,
  onClick,
}: StatCardProps) {
  return (
    <div
      className={cn(
        "rounded-lg border border-neutral-200 bg-white p-6 shadow-card",
        onClick && "cursor-pointer hover:shadow-card-hover transition-shadow duration-200",
        className
      )}
      onClick={onClick}
    >
      <div className="flex items-start justify-between">
        <p className="text-xs font-medium text-neutral-500 uppercase tracking-wide">{label}</p>
        {Icon && (
          <div className={cn("rounded-md p-1.5 bg-neutral-50", iconColor)}>
            <Icon className="h-4 w-4" />
          </div>
        )}
      </div>
      <p className="mt-3 text-3xl font-semibold text-neutral-900 tabular-nums">{value}</p>
      {(sub || trend) && (
        <div className="mt-1.5 flex items-center gap-2">
          {sub && <p className="text-sm text-neutral-500">{sub}</p>}
          {trend && (
            <span
              className={cn(
                "text-xs font-medium",
                trend.value >= 0 ? "text-success" : "text-error"
              )}
            >
              {trend.value >= 0 ? "↑" : "↓"} {Math.abs(trend.value)}% {trend.label}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
