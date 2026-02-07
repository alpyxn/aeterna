import { useState, useEffect } from 'react';

export default function SecurityBanner() {
    const [isInsecure, setIsInsecure] = useState(false);
    const [dismissed, setDismissed] = useState(false);

    useEffect(() => {
        // Check if connection is not secure (HTTP instead of HTTPS)
        // Also check if it's not localhost (localhost is okay for development)
        const isLocalhost = window.location.hostname === 'localhost' ||
            window.location.hostname === '127.0.0.1';
        const isSecure = window.location.protocol === 'https:';

        setIsInsecure(!isSecure && !isLocalhost);
    }, []);

    if (!isInsecure || dismissed) {
        return null;
    }

    return (
        <div className="fixed top-0 left-0 right-0 z-50 bg-gradient-to-r from-red-600 to-orange-600 text-white py-2 px-4 shadow-lg">
            <div className="container mx-auto flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <svg
                        className="w-5 h-5 flex-shrink-0"
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
                    <span className="text-sm font-medium">
                        <span className="font-bold">Güvenli Değil:</span> Bu bağlantı şifrelenmemiş (HTTP).
                        Hassas verileriniz risk altında olabilir.
                    </span>
                </div>
                <button
                    onClick={() => setDismissed(true)}
                    className="text-white/80 hover:text-white transition-colors p-1"
                    aria-label="Kapat"
                >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                </button>
            </div>
        </div>
    );
}
