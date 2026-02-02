import { useState, useEffect, useCallback } from 'react';
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogTitle, AlertDialogDescription, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog"
import { Mail, Clock, Loader2, Trash2, Heart, AlertCircle, RefreshCw, Inbox } from 'lucide-react';
import { apiRequest } from "@/lib/api";

export default function Dashboard({ masterKey }) {
    const [messages, setMessages] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [refreshing, setRefreshing] = useState(false);
    const [actionLoading, setActionLoading] = useState(null);
    const [, setTick] = useState(0);

    const fetchMessages = useCallback(async () => {
        try {
            const data = await apiRequest('/messages', {
                headers: { 'X-Master-Key': masterKey }
            });
            setMessages(data || []);
            setError(null);
        } catch (e) {
            setError(e.message);
        } finally {
            setLoading(false);
        }
    }, [masterKey]);

    useEffect(() => {
        fetchMessages();
        const interval = setInterval(fetchMessages, 30000);
        return () => clearInterval(interval);
    }, [fetchMessages]);

    const handleRefresh = async () => {
        setRefreshing(true);
        await fetchMessages();
        setRefreshing(false);
    };

    // Update countdown every second
    useEffect(() => {
        const timer = setInterval(() => setTick(t => t + 1), 1000);
        return () => clearInterval(timer);
    }, []);

    const handleHeartbeat = async (message) => {
        if (message.status === 'triggered') return;

        setActionLoading(message.id);
        try {
            await apiRequest('/heartbeat', {
                method: 'POST',
                headers: {
                    'X-Master-Key': masterKey
                },
                body: JSON.stringify({ id: message.id })
            });
            await fetchMessages();
        } catch (e) {
            setError(e.message);
        } finally {
            setActionLoading(null);
        }
    };

    const handleDelete = async (message) => {
        setActionLoading(message.id);
        try {
            await apiRequest(`/messages/${message.id}`, {
                method: 'DELETE',
                headers: { 'X-Master-Key': masterKey }
            });
            await fetchMessages();
        } catch (e) {
            setError(e.message);
        } finally {
            setActionLoading(null);
        }
    };

    const formatTimeRemaining = (message) => {
        if (message.status === 'triggered') {
            return { text: 'TRIGGERED', className: 'text-red-500' };
        }

        const lastSeen = new Date(message.last_seen);
        const triggerTime = new Date(lastSeen.getTime() + message.trigger_duration * 60 * 1000);
        const now = new Date();
        const remaining = triggerTime - now;

        if (remaining <= 0) {
            return { text: 'TRIGGERING...', className: 'text-red-500 animate-pulse' };
        }

        const hours = Math.floor(remaining / (1000 * 60 * 60));
        const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((remaining % (1000 * 60)) / 1000);

        if (hours > 24) {
            const days = Math.floor(hours / 24);
            return { text: `${days}d ${hours % 24}h`, className: 'text-green-400' };
        } else if (hours > 0) {
            return { text: `${hours}h ${minutes}m`, className: hours < 2 ? 'text-yellow-400' : 'text-green-400' };
        } else if (minutes > 0) {
            return { text: `${minutes}m ${seconds}s`, className: minutes < 10 ? 'text-orange-400' : 'text-yellow-400' };
        } else {
            return { text: `${seconds}s`, className: 'text-red-400 animate-pulse' };
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center py-20">
                <Loader2 className="w-8 h-8 animate-spin text-cyan-400" />
            </div>
        );
    }

    return (
        <div className="w-full max-w-6xl space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-black text-white">Control Center</h1>
                    <p className="text-slate-500 text-sm">{messages.length} active switch{messages.length !== 1 ? 'es' : ''}</p>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    className="border-slate-700"
                    onClick={handleRefresh}
                    disabled={loading || refreshing}
                >
                    <RefreshCw className={`w-4 h-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
                    Refresh
                </Button>
            </div>

            {error && (
                <Alert variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                        <div className="flex items-center justify-between gap-4">
                            <span>{error}</span>
                            <Button
                                variant="outline"
                                size="sm"
                                className="border-red-500/40 hover:bg-red-500/10"
                                onClick={handleRefresh}
                                disabled={loading || refreshing}
                            >
                                Retry
                            </Button>
                        </div>
                    </AlertDescription>
                </Alert>
            )}

            {messages.length === 0 ? (
                <Card className="border-slate-800 bg-slate-900/50">
                    <CardContent className="py-12 text-center space-y-3">
                        <div className="mx-auto w-10 h-10 rounded-full bg-cyan-500/10 flex items-center justify-center">
                            <Inbox className="w-5 h-5 text-cyan-400" />
                        </div>
                        <p className="text-slate-400 font-medium">No active switches</p>
                        <p className="text-slate-600 text-sm">Create one to get started.</p>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    {messages.map(message => {
                        const timeInfo = formatTimeRemaining(message);
                        const isTriggered = message.status === 'triggered';

                        return (
                            <Card key={message.id} className={`glowing-card border-slate-800 ${isTriggered ? 'border-red-500/30 bg-red-500/5' : ''}`}>
                                <CardHeader className="pb-4">
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex-1 space-y-1">
                                            <div className="flex items-center gap-2">
                                                <Mail className="w-4 h-4 text-cyan-400" />
                                                <CardTitle className="text-base truncate">{message.recipient_email}</CardTitle>
                                            </div>
                                            <CardDescription className="text-xs">
                                                Created {new Date(message.created_at).toLocaleDateString()}
                                            </CardDescription>
                                        </div>
                                        <div className={`text-[10px] px-2 py-1 rounded-full font-semibold ${isTriggered ? 'bg-red-500/20 text-red-400' : 'bg-emerald-500/20 text-emerald-300'}`}>
                                            {isTriggered ? 'Triggered' : 'Active'}
                                        </div>
                                    </div>
                                </CardHeader>
                                <CardContent className="space-y-4 pb-4">
                                    <div className="bg-slate-950/50 p-3 rounded border border-slate-800/50">
                                        <div className="text-[10px] text-slate-500 uppercase tracking-wider mb-1">Message</div>
                                        <div className="text-xs text-slate-300 line-clamp-2">
                                            {message.content?.substring(0, 120)}{message.content?.length > 120 ? '...' : ''}
                                        </div>
                                    </div>
                                    <div className="grid grid-cols-2 gap-2">
                                        <div className="bg-slate-950/50 p-3 rounded border border-slate-800/50">
                                            <div className="text-[10px] text-slate-500 uppercase tracking-wider mb-1">Last Heartbeat</div>
                                            <div className="text-xs font-medium text-slate-300">
                                                {new Date(message.last_seen).toLocaleString('en-US', {
                                                    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
                                                })}
                                            </div>
                                        </div>
                                        <div className="bg-slate-950/50 p-3 rounded border border-slate-800/50">
                                            <div className="text-[10px] text-slate-500 uppercase tracking-wider mb-1">Time Remaining</div>
                                            <div className={`text-sm font-bold ${timeInfo.className}`}>
                                                {timeInfo.text}
                                            </div>
                                        </div>
                                    </div>
                                </CardContent>
                                <CardFooter className="flex gap-2 pt-3">
                                    <Button
                                        className={`flex-1 text-sm h-9 ${isTriggered ? 'bg-slate-700 text-slate-400 cursor-not-allowed' : 'bg-cyan-600 hover:bg-cyan-500 text-white'}`}
                                        onClick={() => handleHeartbeat(message)}
                                        disabled={actionLoading === message.id || isTriggered}
                                    >
                                        {actionLoading === message.id ? (
                                            <Loader2 className="w-4 h-4 animate-spin" />
                                        ) : isTriggered ? (
                                            <><AlertCircle className="w-3 h-3 mr-1" /> Delivered</>
                                        ) : (
                                            <><Heart className="w-3 h-3 mr-1" /> I'm Alive</>
                                        )}
                                    </Button>
                                    <AlertDialog>
                                        <AlertDialogTrigger asChild>
                                            <Button
                                                variant="outline"
                                                size="icon"
                                                className="h-9 w-9 border-slate-700 hover:bg-red-500/10 hover:border-red-500/50 hover:text-red-400"
                                                disabled={actionLoading === message.id}
                                            >
                                                <Trash2 className="w-4 h-4" />
                                            </Button>
                                        </AlertDialogTrigger>
                                        <AlertDialogContent>
                                            <AlertDialogTitle>Delete Switch?</AlertDialogTitle>
                                            <AlertDialogDescription>
                                                This will permanently delete this switch. The message will not be delivered.
                                            </AlertDialogDescription>
                                            <div className="flex justify-end gap-2 mt-4">
                                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                                <AlertDialogAction onClick={() => handleDelete(message)}>
                                                    Delete
                                                </AlertDialogAction>
                                            </div>
                                        </AlertDialogContent>
                                    </AlertDialog>
                                </CardFooter>
                            </Card>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
