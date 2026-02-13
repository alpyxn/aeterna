import { useEffect, useState, useMemo } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Lock, ChevronRight, Loader2, Check, X } from 'lucide-react';
import { apiRequest } from "@/lib/api";

const passwordRules = [
    { id: 'length', label: 'At least 8 characters', test: (p) => p.length >= 8 },
    { id: 'upper', label: 'One uppercase letter', test: (p) => /[A-Z]/.test(p) },
    { id: 'lower', label: 'One lowercase letter', test: (p) => /[a-z]/.test(p) },
    { id: 'number', label: 'One number', test: (p) => /[0-9]/.test(p) },
    { id: 'special', label: 'One special character (!@#$%^&*)', test: (p) => /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(p) },
];

export default function VaultLock({ onUnlock }) {
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [ownerEmail, setOwnerEmail] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [configured, setConfigured] = useState(null);

    useEffect(() => {
        const checkConfigured = async () => {
            try {
                const data = await apiRequest('/setup/status');
                setConfigured(Boolean(data?.configured));
            } catch (e) {
                // If API is unavailable (e.g., 502 Bad Gateway), default to false
                // This allows users to set up their password even if backend was temporarily unavailable
                setConfigured(false);
                
                // Show helpful error message for backend unavailability
                const errorMessage = e.message || '';
                if (errorMessage.includes('502') || errorMessage.includes('Bad Gateway') || 
                    errorMessage.includes('Failed to fetch') || errorMessage.includes('NetworkError')) {
                    setError('Backend service is unavailable. Please ensure the backend is running and try again.');
                }
            }
        };
        checkConfigured();
    }, []);

    const passwordStrength = useMemo(() => {
        const passed = passwordRules.filter(rule => rule.test(password));
        return {
            passed,
            score: passed.length,
            isValid: passed.length === passwordRules.length,
            percentage: (passed.length / passwordRules.length) * 100,
        };
    }, [password]);

    const strengthLabel = useMemo(() => {
        if (password.length === 0) return { text: '', color: '' };
        if (passwordStrength.score <= 2) return { text: 'Weak', color: 'bg-red-500' };
        if (passwordStrength.score <= 3) return { text: 'Fair', color: 'bg-orange-500' };
        if (passwordStrength.score <= 4) return { text: 'Good', color: 'bg-yellow-500' };
        return { text: 'Strong', color: 'bg-teal-500' };
    }, [password, passwordStrength.score]);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError('');

        try {
            if (configured === false) {
                if (!passwordStrength.isValid) {
                    setError('Please meet all password requirements.');
                    setLoading(false);
                    return;
                }
                if (password !== confirmPassword) {
                    setError('Passwords do not match.');
                    setLoading(false);
                    return;
                }
                await apiRequest('/setup', {
                    method: 'POST',
                    body: JSON.stringify({ password, owner_email: ownerEmail })
                });
                onUnlock();
            } else {
                await apiRequest('/auth/verify', {
                    method: 'POST',
                    body: JSON.stringify({ password })
                });
                onUnlock();
            }
        } catch (e) {
            const errorMessage = e.message || '';
            // Provide helpful error messages for common issues
            if (errorMessage.includes('502') || errorMessage.includes('Bad Gateway')) {
                setError('Backend service is unavailable. Please check that the backend container is running.');
            } else if (errorMessage.includes('Failed to fetch') || errorMessage.includes('NetworkError')) {
                setError('Cannot connect to backend. Please ensure the backend service is running.');
            } else if (errorMessage.includes('already_configured')) {
                setError('Master password is already configured. Please use the login form.');
                setConfigured(true);
            } else {
                setError(errorMessage || 'Invalid master credentials.');
            }
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="w-full max-w-md">
            <Card className="glowing-card">
                <CardHeader className="text-center pb-4">
                    <div className="mx-auto w-12 h-12 bg-dark-800 rounded-xl flex items-center justify-center mb-4">
                        <Lock className="w-5 h-5 text-teal-400" />
                    </div>
                    <CardTitle className="text-xl font-semibold text-dark-100">Aeterna Vault</CardTitle>
                    <CardDescription className="text-dark-400">
                        {configured === null
                            ? 'Checking security status...'
                            : configured === false
                                ? 'Set a master password to secure your control center.'
                                : 'Enter your master password to continue.'}
                    </CardDescription>
                </CardHeader>
                <form onSubmit={handleSubmit}>
                    <CardContent className="space-y-4">
                        <div className="space-y-2">
                            <Input
                                type="password"
                                placeholder={configured === false ? 'Create Master Password' : 'Enter Master Password'}
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className="bg-dark-950 border-dark-700 h-11 text-center tracking-widest focus:border-teal-500 text-dark-100 placeholder:text-dark-500"
                                autoFocus
                            />
                        </div>
                        {configured === false && (
                            <>
                                <div className="space-y-2">
                                    <Input
                                        type="password"
                                        placeholder="Confirm Master Password"
                                        value={confirmPassword}
                                        onChange={(e) => setConfirmPassword(e.target.value)}
                                        className="bg-dark-950 border-dark-700 h-11 text-center tracking-widest focus:border-teal-500 text-dark-100 placeholder:text-dark-500"
                                    />
                                </div>

                                <div className="space-y-2">
                                    <label className="text-xs font-medium text-dark-400">Your Email (for reminders)</label>
                                    <Input
                                        type="email"
                                        placeholder="your@email.com"
                                        value={ownerEmail}
                                        onChange={(e) => setOwnerEmail(e.target.value)}
                                        className="bg-dark-950 border-dark-700 h-11 focus:border-teal-500 text-dark-100 placeholder:text-dark-500"
                                    />
                                    <p className="text-xs text-dark-500">You'll receive reminder emails when any switch needs attention</p>
                                </div>

                                {/* Password Strength Indicator */}
                                {password.length > 0 && (
                                    <div className="space-y-3">
                                        <div className="space-y-1.5">
                                            <div className="flex justify-between text-xs">
                                                <span className="text-dark-500">Password Strength</span>
                                                <span className={`font-medium ${strengthLabel.text === 'Strong' ? 'text-teal-400' :
                                                    strengthLabel.text === 'Good' ? 'text-yellow-400' :
                                                        strengthLabel.text === 'Fair' ? 'text-orange-400' :
                                                            'text-red-400'
                                                    }`}>{strengthLabel.text}</span>
                                            </div>
                                            <div className="h-1 bg-dark-800 rounded-full overflow-hidden">
                                                <div
                                                    className={`h-full transition-all duration-300 ${strengthLabel.color}`}
                                                    style={{ width: `${passwordStrength.percentage}%` }}
                                                />
                                            </div>
                                        </div>

                                        <div className="grid grid-cols-1 gap-1.5 pt-1">
                                            {passwordRules.map(rule => {
                                                const passed = rule.test(password);
                                                return (
                                                    <div
                                                        key={rule.id}
                                                        className={`flex items-center gap-2 text-xs transition-colors ${passed ? 'text-teal-400' : 'text-dark-500'
                                                            }`}
                                                    >
                                                        {passed ? (
                                                            <Check className="w-3 h-3" />
                                                        ) : (
                                                            <X className="w-3 h-3" />
                                                        )}
                                                        <span>{rule.label}</span>
                                                    </div>
                                                );
                                            })}
                                        </div>
                                    </div>
                                )}
                            </>
                        )}
                        {error && (
                            <p className="text-xs text-red-400 text-center">{error}</p>
                        )}
                    </CardContent>
                    <CardFooter>
                        <Button
                            className="w-full h-11 bg-teal-600 hover:bg-teal-500 text-white font-medium"
                            type="submit"
                            disabled={loading || configured === null || !password || (configured === false && !confirmPassword)}
                        >
                            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : (configured === false ? "Set Password" : "Unlock")}
                            {!loading && <ChevronRight className="w-4 h-4 ml-2" />}
                        </Button>
                    </CardFooter>
                </form>
            </Card>
            <p className="mt-6 text-center text-xs text-dark-500">
                Authorized access only
            </p>
        </div>
    );
}
