// src/components/ui/Card.tsx

import { cn } from "@/utils";

interface CardProps {
  children: React.ReactNode;
  className?: string;
  // hover lift effect — useful for item cards and order cards
  hoverable?: boolean;
  onClick?: () => void;
}

export function Card({ children, className, hoverable, onClick }: CardProps) {
  return (
    <div
      onClick={onClick}
      className={cn(
        "bg-white dark:bg-stone-900",
        "border border-stone-200 dark:border-stone-800",
        "rounded-2xl",
        "shadow-sm",
        hoverable &&
          "cursor-pointer transition-all duration-200 hover:shadow-md hover:-translate-y-0.5 active:translate-y-0",
        onClick && "cursor-pointer",
        className
      )}
    >
      {children}
    </div>
  );
}

// Convenience sub-components for consistent internal padding

export function CardHeader({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "px-5 pt-5 pb-4 border-b border-stone-100 dark:border-stone-800",
        className
      )}
    >
      {children}
    </div>
  );
}

export function CardBody({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("px-5 py-4", className)}>{children}</div>;
}

export function CardFooter({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "px-5 pb-5 pt-4 border-t border-stone-100 dark:border-stone-800",
        className
      )}
    >
      {children}
    </div>
  );
}