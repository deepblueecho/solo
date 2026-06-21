import { cn } from "@/lib/utils";

export function panelHeaderClass(className?: string) {
  return cn(
    "flex items-center justify-between border-b-2 border-black bg-brutal-cream px-4 py-3",
    className,
  );
}

export function panelTitleClass(className?: string) {
  return cn("font-heading text-sm font-black uppercase tracking-wider", className);
}
