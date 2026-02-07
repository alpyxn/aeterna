import { useState, useEffect, useCallback } from 'react';
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogTitle, AlertDialogDescription, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog"
import { Mail, Clock, Loader2, Trash2, Heart, AlertCircle, RefreshCw, Inbox, Eye, Pencil } from 'lucide-react';
import { apiRequest } from "@/lib/api";
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { Select } from "@/components/ui/select"

export default function Dashboard() {
    const [messages, setMessages] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [refreshing, setRefreshing] = useState(false);
    const [actionLoading, setActionLoading] = useState(null);
    const [, setTick] = useState(0);

    const fetchMessages = useCallback(async () => {
        try {
            const data = await apiRequest('/messages');
            setMessages(data || []);
            setError(null);
        } catch (e) {
            setError(e.message);
        } finally {
            setLoading(false);
        }
    }, []);

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
                method: 'DELETE'
            });
            await fetchMessages();
        } catch (e) {
            setError(e.message);
        } finally {
            setActionLoading(null);
        }
    };

    // Edit state
    const [editingMessage, setEditingMessage] = useState(null);
    const [editContent, setEditContent] = useState('');
    const [editDuration, setEditDuration] = useState(1440);
    const [editDialogOpen, setEditDialogOpen] = useState(false);

    const timePresets = [
        { label: '1 Minute (Debug)', value: 1 },
        { label: '15 Minutes (Test)', value: 15 },
        { label: '1 Hour', value: 60 },
        { label: '1 Day', value: 1440 },
        { label: '3 Days', value: 4320 },
        { label: '1 Week', value: 10080 },
        { label: '2 Weeks', value: 20160 },
        { label: '1 Month', value: 43200 },
        { label: '3 Months', value: 129600 },
        { label: '6 Months', value: 259200 },
        { label: '1 Year', value: 525600 },
    ];

    const openEditDialog = (message) => {
        setEditingMessage(message);
        setEditContent(message.content);
        setEditDuration(message.trigger_duration);
        setEditDialogOpen(true);
    };

    const handleUpdate = async () => {
        if (!editingMessage) return;

        setActionLoading(editingMessage.id);
        try {
            await apiRequest(`/messages/${editingMessage.id}`, {
                method: 'PUT',
                body: JSON.stringify({
                    content: editContent,
                    trigger_duration: editDuration
                })
            });
            setEditDialogOpen(false);
            setEditingMessage(null);
            await fetchMessages();
        } catch (e) {
            setError(e.message);
        } finally {
            setActionLoading(null);
        }
    };

    const formatTimeRemaining = (message) => {
        if (message.status === 'triggered') {
            return { text: 'TRIGGERED', className: 'text-red-400' };
        }

        const lastSeen = new Date(message.last_seen);
        const triggerTime = new Date(lastSeen.getTime() + message.trigger_duration * 60 * 1000);
        const now = new Date();
        const remaining = triggerTime - now;

        if (remaining <= 0) {
            return { text: 'TRIGGERING...', className: 'text-red-400 animate-pulse' };
        }

        const hours = Math.floor(remaining / (1000 * 60 * 60));
        const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((remaining % (1000 * 60)) / 1000);

        if (hours > 24) {
            const days = Math.floor(hours / 24);
            return { text: `${days}d ${hours % 24}h`, className: 'text-teal-400' };
        } else if (hours > 0) {
            return { text: `${hours}h ${minutes}m`, className: hours < 2 ? 'text-yellow-400' : 'text-teal-400' };
        } else if (minutes > 0) {
            return { text: `${minutes}m ${seconds}s`, className: minutes < 10 ? 'text-orange-400' : 'text-yellow-400' };
        } else {
            return { text: `${seconds}s`, className: 'text-red-400 animate-pulse' };
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center py-20">
                <Loader2 className="w-6 h-6 animate-spin text-teal-400" />
            </div>
        );
    }

    return (
        <div className="w-full max-w-6xl space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-semibold text-dark-100">Control Center</h1>
                    <p className="text-dark-400 text-sm">{messages.length} switch{messages.length !== 1 ? 'es' : ''}</p>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    className="border-dark-700 text-dark-300 hover:bg-dark-800 hover:text-dark-100"
                    onClick={handleRefresh}
                    disabled={loading || refreshing}
                >
                    <RefreshCw className={`w-4 h-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
                    Refresh
                </Button>
            </div>

            {error && (
                <Alert variant="destructive" className="border-red-500/20 bg-red-500/10">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                        <div className="flex items-center justify-between gap-4">
                            <span>{error}</span>
                            <Button
                                variant="outline"
                                size="sm"
                                className="border-red-500/30 hover:bg-red-500/10"
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
                <Card className="glowing-card">
                    <CardContent className="py-12 text-center space-y-3">
                        <div className="mx-auto w-10 h-10 rounded-full bg-dark-800 flex items-center justify-center">
                            <Inbox className="w-5 h-5 text-dark-400" />
                        </div>
                        <p className="text-dark-300 font-medium">No switches yet</p>
                        <p className="text-dark-500 text-sm">Create one to get started.</p>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    {messages.map(message => {
                        const timeInfo = formatTimeRemaining(message);
                        const isTriggered = message.status === 'triggered';

                        return (
                            <Card key={message.id} className={`glowing-card ${isTriggered ? 'border-red-500/30' : ''}`}>
                                <CardHeader className="pb-3">
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex-1 space-y-1">
                                            <div className="flex items-center gap-2">
                                                <Mail className="w-4 h-4 text-teal-400" />
                                                <CardTitle className="text-sm font-medium truncate">{message.recipient_email}</CardTitle>
                                            </div>
                                            <CardDescription className="text-xs text-dark-500">
                                                Created {new Date(message.created_at).toLocaleDateString()}
                                            </CardDescription>
                                        </div>
                                        <div className={`text-[10px] px-2 py-1 rounded-full font-medium ${isTriggered ? 'bg-red-500/10 text-red-400' : 'bg-teal-500/10 text-teal-400'}`}>
                                            {isTriggered ? 'Triggered' : 'Active'}
                                        </div>
                                    </div>
                                </CardHeader>
                                <CardContent className="space-y-3 pb-3">
                                    <div className="bg-dark-950 p-3 rounded-lg border border-dark-800 space-y-2">
                                        <div className="text-[10px] text-dark-500 uppercase tracking-wider">Message</div>
                                        <div className="relative">
                                            <div className="text-xs text-dark-300 line-clamp-2">
                                                {message.content?.substring(0, 120)}{message.content?.length > 120 ? '...' : ''}
                                            </div>
                                            {message.content?.length > 120 && (
                                                <div className="absolute inset-x-0 bottom-0 h-4 bg-gradient-to-t from-dark-950 to-transparent pointer-events-none" />
                                            )}
                                        </div>

                                        {message.content?.length > 120 && (
                                            <div className="flex justify-start">
                                                <Dialog>
                                                    <DialogTrigger asChild>
                                                        <button className="flex items-center gap-1.5 px-3 py-1 bg-dark-800 border border-dark-600 rounded-lg text-[10px] text-teal-400 font-medium shadow-lg transition-all hover:bg-dark-700 hover:scale-105 active:scale-95">
                                                            <Eye className="w-3 h-3" /> View Full Message
                                                        </button>
                                                    </DialogTrigger>
                                                    <DialogContent className="max-w-[95vw] sm:max-w-2xl max-h-[90vh] overflow-y-auto">
                                                        <DialogHeader>
                                                            <DialogTitle>Message Content</DialogTitle>
                                                            <DialogDescription>
                                                                Recipient: {message.recipient_email}
                                                            </DialogDescription>
                                                        </DialogHeader>
                                                        <div className="mt-4 bg-dark-950 p-4 rounded-lg border border-dark-800 max-h-[60vh] overflow-y-auto whitespace-pre-wrap text-sm text-dark-200">
                                                            {message.content}
                                                        </div>
                                                    </DialogContent>
                                                </Dialog>
                                            </div>
                                        )}
                                    </div>
                                    <div className="grid grid-cols-2 gap-2">
                                        <div className="bg-dark-950 p-3 rounded-lg border border-dark-800">
                                            <div className="text-[10px] text-dark-500 uppercase tracking-wider mb-1">Last Heartbeat</div>
                                            <div className="text-xs font-medium text-dark-300">
                                                {new Date(message.last_seen).toLocaleString('en-US', {
                                                    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
                                                })}
                                            </div>
                                        </div>
                                        <div className="bg-dark-950 p-3 rounded-lg border border-dark-800">
                                            <div className="text-[10px] text-dark-500 uppercase tracking-wider mb-1">Time Remaining</div>
                                            <div className={`text-sm font-semibold ${timeInfo.className}`}>
                                                {timeInfo.text}
                                            </div>
                                        </div>
                                    </div>
                                </CardContent>
                                <CardFooter className="flex gap-2 pt-2">
                                    <Button
                                        className={`flex-1 text-sm h-9 ${isTriggered ? 'bg-dark-800 text-dark-500 cursor-not-allowed' : 'bg-teal-600 hover:bg-teal-500 text-white'}`}
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
                                    {!isTriggered && (
                                        <Button
                                            variant="outline"
                                            size="icon"
                                            className="h-9 w-9 border-dark-700 hover:bg-teal-500/10 hover:border-teal-500/30 hover:text-teal-400"
                                            onClick={() => openEditDialog(message)}
                                            disabled={actionLoading === message.id}
                                        >
                                            <Pencil className="w-4 h-4" />
                                        </Button>
                                    )}
                                    <AlertDialog>
                                        <AlertDialogTrigger asChild>
                                            <Button
                                                variant="outline"
                                                size="icon"
                                                className="h-9 w-9 border-dark-700 hover:bg-red-500/10 hover:border-red-500/30 hover:text-red-400"
                                                disabled={actionLoading === message.id}
                                            >
                                                <Trash2 className="w-4 h-4" />
                                            </Button>
                                        </AlertDialogTrigger>
                                        <AlertDialogContent className="bg-dark-900 border-dark-700">
                                            <AlertDialogTitle>Delete Switch?</AlertDialogTitle>
                                            <AlertDialogDescription className="text-dark-400">
                                                This will permanently delete this switch. The message will not be delivered.
                                            </AlertDialogDescription>
                                            <div className="flex justify-end gap-2 mt-4">
                                                <AlertDialogCancel className="bg-dark-800 border-dark-700 text-dark-200 hover:bg-dark-700">Cancel</AlertDialogCancel>
                                                <AlertDialogAction onClick={() => handleDelete(message)} className="bg-red-600 hover:bg-red-500">
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

            {/* Edit Dialog */}
            <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
                <DialogContent className="max-w-[95vw] sm:max-w-2xl max-h-[90vh] overflow-y-auto bg-dark-900 border-dark-700">
                    <DialogHeader>
                        <DialogTitle>Edit Switch</DialogTitle>
                        <DialogDescription className="text-dark-400">
                            {editingMessage?.recipient_email}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 mt-4">
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400">Message Content</label>
                            <Textarea
                                value={editContent}
                                onChange={(e) => setEditContent(e.target.value)}
                                className="min-h-[150px] bg-dark-950 border-dark-700 focus:border-teal-500 resize-none text-dark-100 placeholder:text-dark-500"
                                placeholder="Enter your message..."
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Clock className="w-3 h-3" /> Trigger After
                            </label>
                            <Select
                                value={editDuration}
                                onChange={(e) => setEditDuration(Number(e.target.value))}
                                className="bg-dark-950 border-dark-700 text-dark-100"
                            >
                                {timePresets.map(preset => (
                                    <option key={preset.value} value={preset.value}>
                                        {preset.label}
                                    </option>
                                ))}
                            </Select>
                            <p className="text-[10px] text-dark-500">
                                Timer will reset when you save changes
                            </p>
                        </div>
                    </div>
                    <div className="flex flex-col-reverse sm:flex-row justify-end gap-2 mt-6">
                        <Button
                            variant="outline"
                            onClick={() => setEditDialogOpen(false)}
                            className="border-dark-700 text-dark-200 hover:bg-dark-800"
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleUpdate}
                            disabled={actionLoading || !editContent.trim()}
                            className="bg-teal-600 hover:bg-teal-500 text-white"
                        >
                            {actionLoading ? (
                                <Loader2 className="w-4 h-4 animate-spin mr-2" />
                            ) : null}
                            Save Changes
                        </Button>
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
}
