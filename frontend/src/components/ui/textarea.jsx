import * as React from "react"

import { cn } from "@/lib/utils"

function Textarea({
  className,
  ...props
}) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "min-h-16 w-full rounded-lg border bg-[#0d1117] border-[#30363d] px-3 py-2 text-sm text-[#f0f6fc] placeholder:text-[#6e7681] transition-colors outline-none resize-none",
        "focus:border-[#14b8a6] focus:ring-1 focus:ring-[#14b8a6]/30",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      {...props} />
  );
}

export { Textarea }
