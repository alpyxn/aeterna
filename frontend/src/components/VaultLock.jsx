import { useEffect, useState } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { ShieldAlert, Lock, ChevronRight, Loader2 } from 'lucide-react';
import { apiRequest } from "@/lib/api";

export default function VaultLock({ onUnlock }) {
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [configured, setConfigured] = useState(null);

    useEffect(() => {
        const checkConfigured = async () => {
            try {
                const data = await apiRequest('/setup/status');
                setConfigured(Boolean(data?.configured));
            } catch (e) {
                setConfigured(true);
            }
        };
        checkConfigured();
    }, []);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError('');

        try {
            if (configured === false) {
                if (password.length < 8) {
                    setError('Master password must be at least 8 characters.');
                    return;
                }
                if (password !== confirmPassword) {
                    setError('Passwords do not match.');
                    return;
                }
                await apiRequest('/setup', {
                    method: 'POST',
                    body: JSON.stringify({ password })
                });
                onUnlock(password);
            } else {
                await apiRequest('/auth/verify', {
                    method: 'POST',
                    body: JSON.stringify({ password })
                });
                onUnlock(password);
            }
        } catch (e) {
            setError(e.message || 'Invalid master credentials.');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="w-full max-w-md animate-in fade-in zoom-in-95 duration-700">
            <Card className="glowing-card border-none">
                <CardHeader className="text-center">
                    <div className="mx-auto w-12 h-12 bg-cyan-500/10 rounded-full flex items-center justify-center mb-4">
                        <Lock className="w-6 h-6 text-cyan-500" />
                    </div>
                    <CardTitle className="text-2xl font-black">AETERNA VAULT</CardTitle>
                    <CardDescription>
                        {configured === null
                            ? 'Checking security status...'
                            : configured === false
                            ? 'Set a master password to secure your control center.'
                            : 'Master authorization required to access control center.'}
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
                                className="bg-slate-950 border-slate-800 h-12 text-center text-lg tracking-widest focus:ring-cyan-500/20"
                                autoFocus
                            />
                        </div>
                        {configured === false && (
                            <div className="space-y-2">
                                <Input
                                    type="password"
                                    placeholder="Confirm Master Password"
                                    value={confirmPassword}
                                    onChange={(e) => setConfirmPassword(e.target.value)}
                                    className="bg-slate-950 border-slate-800 h-12 text-center text-lg tracking-widest focus:ring-cyan-500/20"
                                />
                            </div>
                        )}
                        {error && (
                            <p className="text-xs text-red-400 text-center">{error}</p>
                        )}
                    </CardContent>
                    <CardFooter>
                        <Button
                            className="w-full h-12 bg-cyan-600 hover:bg-cyan-500 text-white font-bold"
                            type="submit"
                            disabled={loading || configured === null || !password || (configured === false && !confirmPassword)}
                        >
                            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : (configured === false ? "SET MASTER PASSWORD" : "UNLOCK CONTROL CENTER")}
                            {!loading && <ChevronRight className="w-4 h-4 ml-2" />}
                        </Button>
                    </CardFooter>
                </form>
            </Card>
            <p className="mt-8 text-center text-[10px] text-slate-600 uppercase tracking-[0.3em]">
                Authorized Personnel Only
            </p>
        </div>
    );
}
