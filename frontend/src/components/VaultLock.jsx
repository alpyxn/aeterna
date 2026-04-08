import { useEffect, useState, useMemo } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Lock, ChevronRight, Loader2, Check, X, Copy, AlertTriangle, LogIn, UserPlus, KeyRound } from 'lucide-react';
import { apiRequest } from "@/lib/api";
import { cn } from "@/lib/utils";

const passwordRules = [
    { id: 'length', label: 'At least 8 characters', test: (p) => p.length >= 8 },
    { id: 'upper', label: 'One uppercase letter', test: (p) => /[A-Z]/.test(p) },
    { id: 'lower', label: 'One lowercase letter', test: (p) => /[a-z]/.test(p) },
    { id: 'number', label: 'One number', test: (p) => /[0-9]/.test(p) },
    { id: 'special', label: 'One special character (!@#$%^&*)', test: (p) => /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(p) },
];

const fieldClass =
    'h-11 rounded-lg border-dark-700/90 bg-dark-950/80 px-3.5 text-left text-sm text-dark-100 placeholder:text-dark-500 shadow-inner shadow-black/30';

function VaultBackdrop({ children }) {
    return (
        <div className="relative w-full max-w-[440px] mx-auto">
            <div
                className="pointer-events-none absolute -inset-x-12 -top-16 -bottom-8 overflow-hidden opacity-90"
                aria-hidden
            >
                <div className="absolute -left-20 -top-8 h-72 w-72 rounded-full bg-teal-500/20 blur-3xl" />
                <div className="absolute -right-24 top-24 h-56 w-56 rounded-full bg-blue-600/15 blur-3xl" />
                <div className="absolute left-1/2 top-1/2 h-40 w-40 -translate-x-1/2 -translate-y-1/2 rounded-full bg-teal-400/5 blur-2xl" />
            </div>
            <div className="relative">{children}</div>
        </div>
    );
}

export default function VaultLock({ onUnlock }) {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [ownerEmail, setOwnerEmail] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [configured, setConfigured] = useState(null);
    const [allowRegistration, setAllowRegistration] = useState(false);
    const [showRecoveryKey, setShowRecoveryKey] = useState('');
    const [isResetMode, setIsResetMode] = useState(false);
    const [isRegisterMode, setIsRegisterMode] = useState(false);
    const [recoveryKeyInput, setRecoveryKeyInput] = useState('');
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        const checkConfigured = async () => {
            try {
                const data = await apiRequest('/setup/status');
                setConfigured(Boolean(data?.configured));
                setAllowRegistration(Boolean(data?.allow_registration));
            } catch (e) {
                setConfigured(false);
                const errorMessage = e.message || '';
                if (errorMessage.includes('502') || errorMessage.includes('Bad Gateway') ||
                    errorMessage.includes('Failed to fetch') || errorMessage.includes('NetworkError')) {
                    setError('Backend service is unavailable. Please ensure the backend is running and try again.');
                }
            }
        };
        checkConfigured();
    }, []);

    useEffect(() => {
        if (!allowRegistration) {
            setIsRegisterMode(false);
        }
    }, [allowRegistration]);

    const showAuthTabs = configured === true && allowRegistration && !isResetMode;

    const headline = useMemo(() => {
        if (configured === null) {
            return { title: 'Aeterna Vault', subtitle: 'Checking security status…' };
        }
        if (isResetMode) {
            return { title: 'Reset password', subtitle: 'Enter your email, recovery key, and choose a new strong password.' };
        }
        if (configured === false) {
            return { title: 'Create your vault', subtitle: 'Set your account email and a strong master password to get started.' };
        }
        if (isRegisterMode) {
            return { title: 'Create an account', subtitle: 'Register a new user with email and a strong master password.' };
        }
        return { title: 'Welcome back', subtitle: 'Sign in with your email and master password.' };
    }, [configured, isResetMode, isRegisterMode]);

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

    const headerIcon = isResetMode ? KeyRound : (configured === false || isRegisterMode) ? UserPlus : LogIn;

    const HeaderIcon = headerIcon;

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError('');

        try {
            if (isResetMode) {
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
                if (!recoveryKeyInput) {
                    setError('Recovery key is required.');
                    setLoading(false);
                    return;
                }
                const res = await apiRequest('/auth/reset-password', {
                    method: 'POST',
                    body: JSON.stringify({
                        email: email.trim(),
                        recovery_key: recoveryKeyInput,
                        new_password: password
                    })
                });
                if (res?.recovery_key) {
                    setShowRecoveryKey(res.recovery_key);
                } else {
                    onUnlock();
                }
            } else if (configured === true && isRegisterMode) {
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
                const accountEmail = email.trim();
                if (!accountEmail) {
                    setError('Email is required.');
                    setLoading(false);
                    return;
                }
                const res = await apiRequest('/auth/register', {
                    method: 'POST',
                    body: JSON.stringify({
                        email: accountEmail,
                        password,
                        owner_email: ownerEmail.trim() || accountEmail
                    })
                });
                if (res?.recovery_key) {
                    setShowRecoveryKey(res.recovery_key);
                } else {
                    onUnlock();
                }
            } else if (configured === false) {
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
                const accountEmail = email.trim() || ownerEmail.trim();
                if (!accountEmail) {
                    setError('Email is required.');
                    setLoading(false);
                    return;
                }
                const res = await apiRequest('/setup', {
                    method: 'POST',
                    body: JSON.stringify({
                        email: accountEmail,
                        password,
                        owner_email: ownerEmail.trim() || accountEmail
                    })
                });
                if (res?.recovery_key) {
                    setShowRecoveryKey(res.recovery_key);
                } else {
                    onUnlock();
                }
            } else {
                if (!email.trim()) {
                    setError('Email is required.');
                    setLoading(false);
                    return;
                }
                await apiRequest('/auth/login', {
                    method: 'POST',
                    body: JSON.stringify({ email: email.trim(), password })
                });
                onUnlock();
            }
        } catch (e) {
            const errorMessage = e.message || '';
            if (errorMessage.includes('502') || errorMessage.includes('Bad Gateway')) {
                setError('Backend service is unavailable. Please check that the backend container is running.');
            } else if (errorMessage.includes('Failed to fetch') || errorMessage.includes('NetworkError')) {
                setError('Cannot connect to backend. Please ensure the backend service is running.');
            } else if (errorMessage.includes('already_configured')) {
                setError('An account already exists. Please use the login form.');
                setConfigured(true);
            } else {
                setError(errorMessage || 'Invalid master credentials.');
            }
        } finally {
            setLoading(false);
        }
    };

    if (showRecoveryKey) {
        return (
            <VaultBackdrop>
                <Card className="overflow-hidden rounded-2xl border border-dark-700/90 bg-gradient-to-b from-dark-900 via-dark-900 to-dark-950 shadow-2xl shadow-black/60 ring-1 ring-white/[0.06]">
                    <div className="h-1 w-full bg-gradient-to-r from-transparent via-red-500/50 to-transparent" aria-hidden />
                    <CardHeader className="space-y-4 px-6 pb-2 pt-8 text-center">
                        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-red-500/20 to-dark-950 ring-1 ring-red-500/25 shadow-lg">
                            <AlertTriangle className="h-6 w-6 text-red-400" />
                        </div>
                        <div className="space-y-1.5">
                            <p className="text-[0.65rem] font-semibold uppercase tracking-[0.25em] text-red-400/90">Important</p>
                            <CardTitle className="text-xl font-semibold tracking-tight text-dark-100">Save your recovery key</CardTitle>
                            <CardDescription className="text-sm leading-relaxed text-dark-400">
                                This is the <strong className="text-red-400">only</strong> way to recover your account if you forget your password. We will not show this key again.
                            </CardDescription>
                        </div>
                    </CardHeader>
                    <CardContent className="space-y-4 px-6">
                        <div className="flex items-center justify-between gap-3 rounded-xl border border-dark-700/80 bg-dark-950/90 p-4 shadow-inner">
                            <code className="break-all font-mono text-sm tracking-wider text-teal-400">{showRecoveryKey}</code>
                            <Button
                                variant="ghost"
                                size="sm"
                                type="button"
                                className="shrink-0 text-dark-400 hover:text-teal-400"
                                onClick={() => {
                                    navigator.clipboard.writeText(showRecoveryKey);
                                    setCopied(true);
                                    setTimeout(() => setCopied(false), 2000);
                                }}
                            >
                                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                            </Button>
                        </div>
                    </CardContent>
                    <CardFooter className="px-6 pb-8">
                        <Button
                            className="h-12 w-full rounded-xl bg-red-600 text-base font-semibold shadow-lg shadow-red-950/40 hover:bg-red-500"
                            onClick={() => onUnlock()}
                        >
                            I have securely saved this key
                        </Button>
                    </CardFooter>
                </Card>
            </VaultBackdrop>
        );
    }

    return (
        <VaultBackdrop>
            <Card className="overflow-hidden rounded-2xl border border-dark-700/90 bg-gradient-to-b from-dark-900 via-dark-900 to-dark-950 shadow-2xl shadow-black/60 ring-1 ring-white/[0.06]">
                <div className="h-1 w-full bg-gradient-to-r from-transparent via-teal-500/50 to-transparent" aria-hidden />
                <CardHeader className="space-y-5 px-6 pb-2 pt-8 text-center">
                    <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-teal-500/15 to-dark-950 ring-1 ring-teal-500/20 shadow-lg">
                        {configured === null ? (
                            <Lock className="h-6 w-6 text-teal-400/80" />
                        ) : (
                            <HeaderIcon className="h-6 w-6 text-teal-400" strokeWidth={1.75} />
                        )}
                    </div>
                    <div className="space-y-1.5">
                        <p className="text-[0.65rem] font-semibold uppercase tracking-[0.25em] text-teal-500/90">Aeterna Vault</p>
                        <CardTitle className="text-2xl font-semibold tracking-tight text-dark-100">{headline.title}</CardTitle>
                        <CardDescription className="text-sm leading-relaxed text-dark-400">{headline.subtitle}</CardDescription>
                    </div>

                    {showAuthTabs && (
                        <div
                            className="flex rounded-xl border border-dark-800 bg-dark-950/90 p-1 shadow-inner"
                            role="tablist"
                            aria-label="Sign in or register"
                        >
                            <button
                                type="button"
                                role="tab"
                                aria-selected={!isRegisterMode}
                                className={cn(
                                    'flex flex-1 items-center justify-center gap-2 rounded-lg py-2.5 text-sm font-medium transition-colors',
                                    !isRegisterMode
                                        ? 'bg-dark-800 text-dark-100 shadow-sm ring-1 ring-dark-600/50'
                                        : 'text-dark-500 hover:text-dark-300'
                                )}
                                onClick={() => {
                                    setIsRegisterMode(false);
                                    setError('');
                                    setPassword('');
                                    setConfirmPassword('');
                                    setOwnerEmail('');
                                }}
                            >
                                <LogIn className="h-4 w-4 opacity-80" />
                                Sign in
                            </button>
                            <button
                                type="button"
                                role="tab"
                                aria-selected={isRegisterMode}
                                className={cn(
                                    'flex flex-1 items-center justify-center gap-2 rounded-lg py-2.5 text-sm font-medium transition-colors',
                                    isRegisterMode
                                        ? 'bg-dark-800 text-dark-100 shadow-sm ring-1 ring-dark-600/50'
                                        : 'text-dark-500 hover:text-dark-300'
                                )}
                                onClick={() => {
                                    setIsRegisterMode(true);
                                    setError('');
                                    setPassword('');
                                    setConfirmPassword('');
                                    setOwnerEmail('');
                                }}
                            >
                                <UserPlus className="h-4 w-4 opacity-80" />
                                Register
                            </button>
                        </div>
                    )}
                </CardHeader>
                {configured === null ? (
                    <CardContent className="flex flex-col items-center justify-center gap-3 px-6 py-16">
                        <Loader2 className="h-10 w-10 animate-spin text-teal-500/80" aria-hidden />
                        <p className="text-sm text-dark-500">Loading…</p>
                    </CardContent>
                ) : (
                <form onSubmit={handleSubmit}>
                    <CardContent className="space-y-4 px-6">
                        {configured === true && !isResetMode && !isRegisterMode && (
                            <div className="space-y-2">
                                <label className="text-xs font-medium text-dark-300" htmlFor="vault-email">Email</label>
                                <Input
                                    id="vault-email"
                                    type="email"
                                    autoComplete="username"
                                    placeholder="you@example.com"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    className={fieldClass}
                                />
                            </div>
                        )}
                        {(configured === false || isRegisterMode) && !isResetMode && (
                            <div className="space-y-2">
                                <label className="text-xs font-medium text-dark-300" htmlFor="vault-account-email">Account email</label>
                                <Input
                                    id="vault-account-email"
                                    type="email"
                                    autoComplete="email"
                                    placeholder="you@example.com"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    className={fieldClass}
                                />
                            </div>
                        )}
                        {isResetMode && (
                            <div className="space-y-3">
                                <div className="space-y-2">
                                    <label className="text-xs font-medium text-dark-300" htmlFor="vault-reset-email">Account email</label>
                                    <Input
                                        id="vault-reset-email"
                                        type="email"
                                        autoComplete="username"
                                        placeholder="you@example.com"
                                        value={email}
                                        onChange={(e) => setEmail(e.target.value)}
                                        className={fieldClass}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-xs font-medium text-dark-300" htmlFor="vault-recovery-key">Recovery key</label>
                                    <Input
                                        id="vault-recovery-key"
                                        type="text"
                                        placeholder="RK-…"
                                        value={recoveryKeyInput}
                                        onChange={(e) => setRecoveryKeyInput(e.target.value)}
                                        className={cn(fieldClass, 'font-mono text-sm')}
                                        autoFocus
                                    />
                                </div>
                            </div>
                        )}
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-300" htmlFor="vault-password">
                                {isResetMode ? 'New password' : (configured === false || isRegisterMode) ? 'Password' : 'Master password'}
                            </label>
                            <Input
                                id="vault-password"
                                type="password"
                                autoComplete={(configured === false || isRegisterMode) ? 'new-password' : 'current-password'}
                                placeholder={isResetMode ? 'New password' : (configured === false || isRegisterMode) ? 'Create a strong password' : 'Enter your password'}
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                className={cn(fieldClass, 'font-mono')}
                                autoFocus={!isResetMode && !(configured === true && isRegisterMode)}
                            />
                        </div>
                        {(configured === false || isRegisterMode || isResetMode) && (
                            <>
                                <div className="space-y-2">
                                    <label className="text-xs font-medium text-dark-300" htmlFor="vault-confirm">
                                        {isResetMode ? 'Confirm new password' : 'Confirm password'}
                                    </label>
                                    <Input
                                        id="vault-confirm"
                                        type="password"
                                        placeholder="Repeat password"
                                        value={confirmPassword}
                                        onChange={(e) => setConfirmPassword(e.target.value)}
                                        className={cn(fieldClass, 'font-mono')}
                                    />
                                </div>

                                {!isResetMode && (
                                    <div className="space-y-2">
                                        <label className="text-xs font-medium text-dark-300" htmlFor="vault-owner-email">Reminder email (optional)</label>
                                        <Input
                                            id="vault-owner-email"
                                            type="email"
                                            placeholder="Defaults to account email"
                                            value={ownerEmail}
                                            onChange={(e) => setOwnerEmail(e.target.value)}
                                            className={fieldClass}
                                        />
                                        <p className="text-xs leading-relaxed text-dark-500">
                                            Heartbeat and reminder messages use this address; leave blank to use your account email.
                                        </p>
                                    </div>
                                )}

                                {password.length > 0 && (
                                    <div className="rounded-xl border border-dark-800/90 bg-dark-950/50 p-4">
                                        <div className="space-y-3">
                                            <div className="flex items-center justify-between text-xs">
                                                <span className="font-medium text-dark-400">Password strength</span>
                                                <span className={cn(
                                                    'font-medium',
                                                    strengthLabel.text === 'Strong' ? 'text-teal-400' :
                                                        strengthLabel.text === 'Good' ? 'text-yellow-400' :
                                                            strengthLabel.text === 'Fair' ? 'text-orange-400' :
                                                                'text-red-400'
                                                )}>{strengthLabel.text || '—'}</span>
                                            </div>
                                            <div className="h-1.5 overflow-hidden rounded-full bg-dark-800">
                                                <div
                                                    className={cn('h-full rounded-full transition-all duration-300', strengthLabel.color || 'bg-dark-600')}
                                                    style={{ width: `${passwordStrength.percentage}%` }}
                                                />
                                            </div>
                                            <ul className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                                                {passwordRules.map(rule => {
                                                    const passed = rule.test(password);
                                                    return (
                                                        <li
                                                            key={rule.id}
                                                            className={cn(
                                                                'flex items-center gap-2 text-xs',
                                                                passed ? 'text-teal-400' : 'text-dark-500'
                                                            )}
                                                        >
                                                            {passed ? <Check className="h-3.5 w-3.5 shrink-0" /> : <X className="h-3.5 w-3.5 shrink-0 opacity-50" />}
                                                            <span>{rule.label}</span>
                                                        </li>
                                                    );
                                                })}
                                            </ul>
                                        </div>
                                    </div>
                                )}
                            </>
                        )}
                        {error && (
                            <div
                                role="alert"
                                className="rounded-xl border border-red-500/35 bg-red-500/[0.07] px-4 py-3 text-center text-sm text-red-300"
                            >
                                {error}
                            </div>
                        )}
                    </CardContent>
                    <CardFooter className="flex-col gap-4 border-t border-dark-800/80 bg-dark-950/30 px-6 pb-8 pt-6">
                        <Button
                            className="h-12 w-full rounded-xl bg-teal-600 text-base font-semibold shadow-lg shadow-teal-950/50 hover:bg-teal-500"
                            type="submit"
                            disabled={
                                loading ||
                                configured === null ||
                                !password ||
                                !email.trim() ||
                                ((configured === false || isRegisterMode || isResetMode) && !confirmPassword) ||
                                (isResetMode && !recoveryKeyInput) ||
                                ((configured === false || isRegisterMode || isResetMode) && !passwordStrength.isValid)
                            }
                        >
                            {loading ? <Loader2 className="h-5 w-5 animate-spin" /> : (
                                <>
                                    {isResetMode ? 'Reset password' : configured === false ? 'Create vault' : isRegisterMode ? 'Create account' : 'Sign in'}
                                    {!loading && <ChevronRight className="ml-2 h-4 w-4" />}
                                </>
                            )}
                        </Button>

                        {configured === true && isRegisterMode && !isResetMode && !showAuthTabs && (
                            <button
                                type="button"
                                className="text-sm text-dark-500 underline-offset-4 hover:text-teal-400 hover:underline"
                                onClick={() => {
                                    setIsRegisterMode(false);
                                    setError('');
                                    setPassword('');
                                    setConfirmPassword('');
                                    setOwnerEmail('');
                                }}
                            >
                                Sign in instead
                            </button>
                        )}

                        {configured === true && !isRegisterMode && (
                            <div className="flex flex-col items-center gap-3 sm:flex-row sm:justify-center sm:gap-6">
                                <button
                                    type="button"
                                    className="text-sm text-dark-500 underline-offset-4 hover:text-teal-400 hover:underline"
                                    onClick={() => {
                                        setIsResetMode(!isResetMode);
                                        setError('');
                                        setPassword('');
                                        setConfirmPassword('');
                                        setRecoveryKeyInput('');
                                    }}
                                >
                                    {isResetMode ? 'Back to sign in' : 'Forgot password?'}
                                </button>
                                {allowRegistration && !isResetMode && !showAuthTabs && (
                                    <button
                                        type="button"
                                        className="text-sm font-medium text-teal-500/90 underline-offset-4 hover:text-teal-400 hover:underline"
                                        onClick={() => {
                                            setIsRegisterMode(true);
                                            setError('');
                                            setPassword('');
                                            setConfirmPassword('');
                                            setOwnerEmail('');
                                        }}
                                    >
                                        Create an account
                                    </button>
                                )}
                            </div>
                        )}
                    </CardFooter>
                </form>
                )}
            </Card>
            <p className="mt-8 text-center text-xs text-dark-600">
                Authorized access only
            </p>
        </VaultBackdrop>
    );
}
