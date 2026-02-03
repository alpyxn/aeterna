import * as React from "react"
import { createContext, useContext, useState } from "react"
import { X } from "lucide-react"

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

const DialogContent = ({ children, className = "" }) => {
    const { open, setOpen } = useContext(DialogContext)

    if (!open) return null

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div className="fixed inset-0 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200" onClick={() => setOpen(false)} />
            <div className={`relative z-50 grid w-full max-w-lg gap-4 border border-dark-800 bg-dark-900 p-6 shadow-2xl sm:rounded-xl animate-in zoom-in-95 duration-200 ${className}`}>
                <button
                    onClick={() => setOpen(false)}
                    className="absolute right-4 top-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus:outline-none disabled:pointer-events-none text-dark-400"
                >
                    <X className="h-4 w-4" />
                    <span className="sr-only">Close</span>
                </button>
                {children}
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
