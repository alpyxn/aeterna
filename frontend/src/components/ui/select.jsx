import * as React from "react"

const Select = React.forwardRef(({ className, children, ...props }, ref) => {
    return (
        <select
            className={`flex h-10 w-full items-center justify-between rounded-lg border bg-[#0d1117] border-[#30363d] px-3 py-2 text-sm text-[#f0f6fc] focus:outline-none focus:border-[#14b8a6] focus:ring-1 focus:ring-[#14b8a6]/30 disabled:cursor-not-allowed disabled:opacity-50 ${className}`}
            ref={ref}
            {...props}
        >
            {children}
        </select>
    )
})
Select.displayName = "Select"

export { Select }
