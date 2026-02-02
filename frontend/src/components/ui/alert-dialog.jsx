import * as React from "react"
import { createContext, useContext, useState } from "react"

const AlertDialogContext = createContext({ open: false, setOpen: () => { } })

const AlertDialog = ({ children }) => {
    const [open, setOpen] = useState(false)

    const content = React.Children.toArray(children).find(
        child => child?.type === AlertDialogContent
    )

    return (
        <AlertDialogContext.Provider value={{ open, setOpen }}>
            {children}
            {open && (
                <div className="fixed inset-0 z-50 flex items-center justify-center">
                    <div className="fixed inset-0 bg-black/80" onClick={() => setOpen(false)} />
                    <div className="relative z-50 grid w-full max-w-lg gap-4 border border-slate-800 bg-slate-950 p-6 shadow-lg sm:rounded-lg">
                        <div className="space-y-4">
                            {content?.props?.children}
                        </div>
                    </div>
                </div>
            )}
        </AlertDialogContext.Provider>
    )
}

const AlertDialogTrigger = ({ asChild, children }) => {
    const { setOpen } = useContext(AlertDialogContext)

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

const AlertDialogContent = ({ children }) => {
    return null
}

const AlertDialogTitle = ({ children }) => {
    return <h2 className="text-lg font-semibold text-slate-100">{children}</h2>
}

const AlertDialogDescription = ({ children }) => {
    return <p className="text-sm text-slate-400">{children}</p>
}

const AlertDialogFooter = ({ children }) => {
    return <div className="flex justify-end gap-2 mt-4">{children}</div>
}

const AlertDialogAction = ({ onClick, children, className = "" }) => {
    const { setOpen } = useContext(AlertDialogContext)
    return (
        <button
            onClick={(e) => {
                onClick?.(e)
                setOpen(false)
            }}
            className={`inline-flex h-10 items-center justify-center rounded-md bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-500 ${className}`}
        >
            {children}
        </button>
    )
}

const AlertDialogCancel = ({ onClick, children }) => {
    const { setOpen } = useContext(AlertDialogContext)
    return (
        <button
            onClick={(e) => {
                onClick?.(e)
                setOpen(false)
            }}
            className="inline-flex h-10 items-center justify-center rounded-md border border-slate-800 bg-transparent px-4 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-800"
        >
            {children}
        </button>
    )
}

export { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogAction, AlertDialogCancel }
