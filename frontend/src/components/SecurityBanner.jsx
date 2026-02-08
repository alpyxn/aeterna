import { useState, useEffect } from 'react';

export default function SecurityBanner() {
    const [isInsecure, setIsInsecure] = useState(false);
    const [dismissed, setDismissed] = useState(() => {
        // Check localStorage for dismissed state
        return localStorage.getItem('security-banner-dismissed') === 'true';
    });

    useEffect(() => {
        // Check if connection is not secure (HTTP instead of HTTPS)
        const isSecure = window.location.protocol === 'https:';
        setIsInsecure(!isSecure);
    }, []);

    const handleDismiss = () => {
        setDismissed(true);
        localStorage.setItem('security-banner-dismissed', 'true');
    };

    if (!isInsecure || dismissed) {
        return null;
    }

    return (
        <>
            {/* Spacer to prevent content overlap */}
            <div className="h-12 sm:h-10" />

            <div className="fixed top-0 left-0 right-0 z-[9999] bg-gradient-to-r from-red-600 to-orange-600 text-white py-2 px-3 sm:px-4 shadow-lg">
                <div className="container mx-auto flex items-center justify-between gap-2 sm:gap-4">
                    <div className="flex items-center gap-2 sm:gap-3 min-w-0">
                        <svg
                            className="w-4 h-4 sm:w-5 sm:h-5 flex-shrink-0"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                        >
                            <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                            />
                        </svg>
                        <span className="text-xs sm:text-sm font-medium truncate sm:whitespace-normal">
                            <span className="font-bold">Not Secure:</span>
                            <span className="hidden sm:inline"> This connection is not encrypted (HTTP). Your sensitive data is not fully protected.</span>
                            <span className="sm:hidden"> HTTP connection!</span>
                        </span>
                    </div>
                    <button
                        onClick={handleDismiss}
                        className="text-white/80 hover:text-white transition-colors p-1 flex-shrink-0"
                        aria-label="Kapat"
                    >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
            </div>
        </>
    );
}
