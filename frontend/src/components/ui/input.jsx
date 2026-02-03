import * as React from "react"

import { cn } from "@/lib/utils"

function Input({
  className,
  type,
  ...props
}) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-10 w-full min-w-0 rounded-lg border bg-[#0d1117] border-[#30363d] px-3 py-2 text-sm text-[#f0f6fc] placeholder:text-[#6e7681] transition-colors outline-none",
        "focus:border-[#14b8a6] focus:ring-1 focus:ring-[#14b8a6]/30",
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      {...props} />
  );
}

export { Input }
