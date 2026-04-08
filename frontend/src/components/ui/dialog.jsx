import * as React from "react"
import { createContext, useContext, useState } from "react"
import { X } from "lucide-react"
import { cn } from "@/lib/utils"

const DialogContext = createContext({ open: false, setOpen: () => { } })

const Dialog = ({ children, open: controlledOpen, onOpenChange }) => {
    const [internalOpen, setInternalOpen] = useState(false)
    const open = controlledOpen !== undefined ? controlledOpen : internalOpen
    const setOpen = onOpenChange !== undefined ? onOpenChange : setInternalOpen

    return (
        <DialogContext.Provider value={{ open, setOpen }}>
            {children}
        </DialogContext.Provider>
    )
}

const DialogTrigger = ({ asChild, children }) => {
    const { setOpen } = useContext(DialogContext)

    if (asChild && React.isValidElement(children)) {
        return React.cloneElement(children, {
            onClick: (e) => {
                children.props.onClick?.(e)
                setOpen(true)
            }
        })
    }

    return <button onClick={() => setOpen(true)}>{children}</button>
}

const DialogContent = ({ children, className = "", contentClassName = "" }) => {
    const { open, setOpen } = useContext(DialogContext)

    if (!open) return null

    return (
        <div
            className="fixed inset-0 z-[200] overflow-y-auto overflow-x-hidden"
            role="presentation"
        >
            <div className="flex min-h-full min-h-[100dvh] items-end justify-center p-0 sm:items-center sm:p-4 sm:py-8">
                <div
                    className="fixed inset-0 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200"
                    onClick={() => setOpen(false)}
                    aria-hidden
                />
                <div
                    className={cn(
                        "relative z-10 my-2 flex w-full max-w-lg flex-col overflow-hidden rounded-t-2xl border border-dark-800 bg-dark-900 shadow-2xl animate-in zoom-in-95 duration-200 sm:my-0 sm:max-h-[min(92dvh,52rem)] sm:rounded-xl",
                        className
                    )}
                >
                    <button
                        type="button"
                        onClick={() => setOpen(false)}
                        className="absolute right-2.5 top-2.5 z-20 rounded-md p-1.5 text-dark-400 opacity-90 ring-offset-dark-900 transition-opacity hover:bg-dark-800 hover:text-dark-100 hover:opacity-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-teal-500/50 sm:right-3 sm:top-3"
                    >
                        <X className="h-4 w-4" />
                        <span className="sr-only">Close</span>
                    </button>
                    <div
                        className={cn(
                            "min-h-0 flex-1 overflow-y-auto overscroll-y-contain p-5 pt-12 sm:p-6 sm:pt-14",
                            contentClassName
                        )}
                    >
                        {children}
                    </div>
                </div>
            </div>
        </div>
    )
}

const DialogHeader = ({ children, className = "" }) => {
    return (
        <div className={`flex flex-col space-y-1.5 text-center sm:text-left ${className}`}>
            {children}
        </div>
    )
}

const DialogTitle = ({ children, className = "" }) => {
    return (
        <h2 className={`text-lg font-semibold text-dark-100 ${className}`}>
            {children}
        </h2>
    )
}

const DialogDescription = ({ children, className = "" }) => {
    return (
        <p className={`text-sm text-dark-400 ${className}`}>
            {children}
        </p>
    )
}

export { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription }
