import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
  {
    variants: {
      variant: {
        default:     "bg-primary-50 text-primary-700",
        success:     "bg-success-light text-success-dark",
        warning:     "bg-warning-light text-warning-dark",
        error:       "bg-error-light text-error-dark",
        neutral:     "bg-neutral-100 text-neutral-600",
        info:        "bg-info-light text-info-dark",
      },
    },
    defaultVariants: { variant: "default" },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
