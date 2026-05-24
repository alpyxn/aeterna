import { useState, useEffect, useCallback, useRef } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogTitle, AlertDialogDescription, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog"
import { Mail, Clock, Loader2, Trash2, Heart, AlertCircle, RefreshCw, Inbox, Eye, Pencil, Paperclip, X, Upload, Plus } from 'lucide-react';
import { apiRequest, uploadFile, deleteAttachment, listAttachments, openEventsStream } from "@/lib/api";
import FarewellLetters from "@/components/FarewellLetters";
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { Select } from "@/components/ui/select"
import { ALLOWED_EXTENSIONS, MAX_FILE_SIZE, MAX_FILES, MAX_TOTAL_SIZE, EMAIL_REGEX, TIME_PRESETS, REMINDER_PRESETS } from "@/lib/constants"
import { formatFileSize, formatMinutes } from "@/lib/formatters"
import { parseRecipientEmails } from "@/lib/parsers"
import { applyDurationToReminders, addReminderValue, removeReminderValue } from "@/lib/reminder-utils"

const RECIPIENT_PREVIEW_LIMIT = 3;

function formatRecipientEmails(value) {
    return parseRecipientEmails(value).join(', ');
}

export default function Dashboard() {
    const [messages, setMessages] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [refreshing, setRefreshing] = useState(false);
    const [actionLoading, setActionLoading] = useState(null);
    const [, setTick] = useState(0);
    const eventRefreshTimeoutRef = useRef(null);
    const fallbackPollIntervalRef = useRef(null);

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

        const stopFallbackPolling = () => {
            if (fallbackPollIntervalRef.current) {
                clearInterval(fallbackPollIntervalRef.current);
                fallbackPollIntervalRef.current = null;
            }
        };

        const startFallbackPolling = () => {
            if (fallbackPollIntervalRef.current) return;
            fallbackPollIntervalRef.current = setInterval(() => {
                void fetchMessages();
            }, 60000);
        };

        const clientId = (globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function')
            ? globalThis.crypto.randomUUID()
            : `web-${Date.now()}-${Math.random().toString(16).slice(2)}`;

        const stream = openEventsStream(clientId);

        const scheduleRefresh = () => {
            if (eventRefreshTimeoutRef.current) return;
            eventRefreshTimeoutRef.current = setTimeout(async () => {
                eventRefreshTimeoutRef.current = null;
                await fetchMessages();
            }, 300);
        };

        const handleEvent = (eventName) => {
            stream.addEventListener(eventName, scheduleRefresh);
        };

        handleEvent('messages.changed');
        handleEvent('attachments.changed');
        handleEvent('farewells.changed');

        stream.onopen = () => {
            stopFallbackPolling();
        };

        stream.onerror = () => {
            // EventSource reconnects automatically; enable fallback polling while disconnected.
            startFallbackPolling();
        };

        // Keep fallback polling active until the SSE stream is confirmed as open.
        startFallbackPolling();

        return () => {
            if (eventRefreshTimeoutRef.current) {
                clearTimeout(eventRefreshTimeoutRef.current);
                eventRefreshTimeoutRef.current = null;
            }
            stopFallbackPolling();
            stream.close();
        };
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
    const [editReminders, setEditReminders] = useState([]);
    const [editDialogOpen, setEditDialogOpen] = useState(false);
    const [editRecipientInput, setEditRecipientInput] = useState('');
    const [editRecipients, setEditRecipients] = useState([]);
    const [editAttachments, setEditAttachments] = useState([]);
    const [editNewFiles, setEditNewFiles] = useState([]);
    const [showEditAttachments, setShowEditAttachments] = useState(false);
    const [editAttachLoading, setEditAttachLoading] = useState(false);
    const editFileInputRef = useRef(null);
    const [queueMessage, setQueueMessage] = useState(null);
    const [queueDialogOpen, setQueueDialogOpen] = useState(false);


    const handleEditDurationChange = (newDuration) => {
        setEditDuration(newDuration);
        setEditReminders((prev) => applyDurationToReminders(prev, newDuration));
    };

    const addEditReminder = (value) => {
        setEditReminders((prev) => addReminderValue(prev, value, editDuration));
    };

    const removeEditReminder = (value) => {
        setEditReminders((prev) => removeReminderValue(prev, value));
    };

    const openEditDialog = async (message) => {
        setEditingMessage(message);
        setEditContent(message.content);
        setEditDuration(message.trigger_duration);
        setEditReminders(message.reminders ? message.reminders.map(r => r.minutes_before) : []);
        setEditRecipients(parseRecipientEmails(message.recipient_email));
        setEditRecipientInput('');
        setEditNewFiles([]);
        setEditDialogOpen(true);
        setShowEditAttachments(message.attachment_count > 0);
        // Load existing attachments
        try {
            const atts = await listAttachments(message.id);
            setEditAttachments(atts || []);
        } catch {
            setEditAttachments([]);
        }
    };

    const openQueueDialog = (message) => {
        setQueueMessage(message);
        setQueueDialogOpen(true);
    };

    const addEditRecipientsFromText = (text) => {
        const parsed = parseRecipientEmails(text);
        if (parsed.length === 0) return { invalid: null };

        let invalid = null;
        setEditRecipients((prev) => {
            const seen = new Set(prev.map((email) => email.toLowerCase()));
            const next = [...prev];
            for (const email of parsed) {
                if (!EMAIL_REGEX.test(email)) {
                    if (!invalid) invalid = email;
                    continue;
                }
                const key = email.toLowerCase();
                if (!seen.has(key)) {
                    seen.add(key);
                    next.push(email);
                }
            }
            return next;
        });

        return { invalid };
    };

    const handleAddEditRecipient = () => {
        const { invalid } = addEditRecipientsFromText(editRecipientInput);
        if (invalid) {
            setError(`Invalid email address: ${invalid}`);
        } else if (error) {
            setError(null);
        }
        setEditRecipientInput('');
    };

    const handleEditRecipientKeyDown = (e) => {
        if (e.key === 'Enter' || e.key === ',' || e.key === ';' || e.key === 'Tab') {
            e.preventDefault();
            if (editRecipientInput.trim()) {
                handleAddEditRecipient();
            }
        }
    };

    const removeEditRecipient = (emailToRemove) => {
        setEditRecipients((prev) => prev.filter((email) => email !== emailToRemove));
    };

    const handleDeleteAttachment = async (attachmentId) => {
        if (!editingMessage) return;
        setEditAttachLoading(true);
        try {
            await deleteAttachment(editingMessage.id, attachmentId);
            setEditAttachments(prev => prev.filter(a => a.id !== attachmentId));
        } catch (e) {
            setError(e.message);
        } finally {
            setEditAttachLoading(false);
        }
    };

    const addEditFiles = (newFiles) => {
        const fileArray = Array.from(newFiles);
        const totalCount = editAttachments.length + editNewFiles.length + fileArray.length;
        if (totalCount > MAX_FILES) {
            setError(`Maximum ${MAX_FILES} files allowed`);
            return;
        }

        const currentSize = editAttachments.reduce((sum, a) => sum + a.size, 0) +
            editNewFiles.reduce((sum, f) => sum + f.size, 0);
        const newSize = fileArray.reduce((sum, f) => sum + f.size, 0);
        if (currentSize + newSize > MAX_TOTAL_SIZE) {
            setError('Total attachment size exceeds 25 MB limit');
            return;
        }

        for (const file of fileArray) {
            const ext = '.' + file.name.split('.').pop().toLowerCase();
            if (!ALLOWED_EXTENSIONS.includes(ext)) {
                setError(`"${file.name}" — type not allowed`);
                return;
            }
            if (file.size > MAX_FILE_SIZE) {
                setError(`"${file.name}" exceeds 10 MB limit`);
                return;
            }
        }

        const existingNames = new Set([
            ...editAttachments.map(a => a.filename),
            ...editNewFiles.map(f => f.name)
        ]);
        const uniqueNew = fileArray.filter(f => !existingNames.has(f.name));
        setEditNewFiles(prev => [...prev, ...uniqueNew]);
    };

    const handleUpdate = async () => {
        if (!editingMessage) return;

        const pendingRecipients = parseRecipientEmails(editRecipientInput);
        const mergedRecipients = [...editRecipients];
        const seen = new Set(mergedRecipients.map((email) => email.toLowerCase()));
        for (const email of pendingRecipients) {
            if (!EMAIL_REGEX.test(email)) {
                setError(`Invalid email address: ${email}`);
                return;
            }
            const key = email.toLowerCase();
            if (!seen.has(key)) {
                seen.add(key);
                mergedRecipients.push(email);
            }
        }
        if (mergedRecipients.length === 0) {
            setError('At least one recipient email is required');
            return;
        }

        setActionLoading(editingMessage.id);
        try {
            await apiRequest(`/messages/${editingMessage.id}`, {
                method: 'PUT',
                body: JSON.stringify({
                    content: editContent,
                    recipient_email: mergedRecipients[0],
                    recipient_emails: mergedRecipients,
                    trigger_duration: editDuration,
                    reminders: editReminders
                })
            });

            // Upload new files
            for (const file of editNewFiles) {
                await uploadFile(editingMessage.id, file);
            }

            setEditDialogOpen(false);
            setEditingMessage(null);
            setEditNewFiles([]);
            setEditRecipientInput('');
            setEditRecipients([]);
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

        const triggerTime = message.next_trigger_at
            ? new Date(message.next_trigger_at)
            : new Date(new Date(message.last_seen).getTime() + message.trigger_duration * 60 * 1000);
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
                        const recipients = parseRecipientEmails(message.recipient_email);
                        const previewRecipients = recipients.slice(0, RECIPIENT_PREVIEW_LIMIT);
                        const hasMoreRecipients = recipients.length > RECIPIENT_PREVIEW_LIMIT;
                        const farewellCount = message.farewell_count || 0;
                        const pendingFarewells = message.pending_farewells || 0;
                        const canManageQueue = isTriggered && farewellCount > 0;
                        const recipientSummaryLabel = recipients.length === 0
                            ? 'No recipients'
                            : recipients.length === 1
                                ? '1 recipient'
                                : `${recipients.length} recipients`;

                        return (
                            <Card key={message.id} className={`glowing-card ${isTriggered ? 'border-red-500/30' : ''}`}>
                                <CardHeader className="pb-3">
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex-1 space-y-1">
                                            <div className="flex items-center gap-2">
                                                <Mail className="w-4 h-4 text-teal-400" />
                                                <CardTitle className="text-sm font-medium truncate">{recipientSummaryLabel}</CardTitle>
                                            </div>

                                            {previewRecipients.length > 0 && (
                                                <div className="flex flex-wrap items-center gap-1.5 pl-6 pr-1 pt-0.5">
                                                    {previewRecipients.map((email) => (
                                                        <span key={`${message.id}-${email}`} className="max-w-[180px] truncate text-[10px] px-2 py-0.5 rounded-full bg-dark-800 text-dark-300 border border-dark-700" title={email}>
                                                            {email}
                                                        </span>
                                                    ))}

                                                    {hasMoreRecipients && (
                                                        <Dialog>
                                                            <DialogTrigger asChild>
                                                                <button className="text-[10px] px-2 py-0.5 rounded-full border border-teal-500/30 text-teal-400 hover:bg-teal-500/10">
                                                                    +{recipients.length - RECIPIENT_PREVIEW_LIMIT} more
                                                                </button>
                                                            </DialogTrigger>
                                                            <DialogContent className="max-w-[95vw] sm:max-w-md max-h-[90vh] overflow-y-auto">
                                                                <DialogHeader>
                                                                    <DialogTitle>All Recipients</DialogTitle>
                                                                    <DialogDescription>
                                                                        This switch has {recipients.length} recipients.
                                                                    </DialogDescription>
                                                                </DialogHeader>
                                                                <div className="mt-2 space-y-1.5">
                                                                    {recipients.map((email, idx) => (
                                                                        <div key={`${message.id}-all-${email}`} className="text-sm text-dark-200 bg-dark-950 border border-dark-800 rounded px-2.5 py-1.5">
                                                                            {idx + 1}. {email}
                                                                        </div>
                                                                    ))}
                                                                </div>
                                                            </DialogContent>
                                                        </Dialog>
                                                    )}
                                                </div>
                                            )}

                                            <CardDescription className="text-xs text-dark-500">
                                                Created {new Date(message.created_at).toLocaleDateString()}
                                            </CardDescription>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {message.attachment_count > 0 && (
                                                <div className="flex items-center gap-1 text-[10px] px-2 py-1 rounded-full bg-dark-800 text-dark-300">
                                                    <Paperclip className="w-3 h-3" />
                                                    {message.attachment_count}
                                                </div>
                                            )}
                                            <div className={`text-[10px] px-2 py-1 rounded-full font-medium ${isTriggered ? 'bg-red-500/10 text-red-400' : 'bg-teal-500/10 text-teal-400'}`}>
                                                {isTriggered ? 'Triggered' : 'Active'}
                                            </div>
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
                                                                Recipients: {formatRecipientEmails(message.recipient_email)}
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
                                    {canManageQueue && (
                                        <Button
                                            variant="outline"
                                            className="h-9 border-dark-700 px-3 text-xs hover:bg-amber-500/10 hover:border-amber-500/30 hover:text-amber-300"
                                            onClick={() => openQueueDialog(message)}
                                            disabled={actionLoading === message.id}
                                        >
                                            <Clock className="w-4 h-4" />
                                            Queue
                                            {pendingFarewells > 0 && (
                                                <span className="ml-1 rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-300">
                                                    {pendingFarewells}
                                                </span>
                                            )}
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
                                                This will permanently delete this switch and all its attachments. The message will not be delivered.
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
                            {formatRecipientEmails(editingMessage?.recipient_email)}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 mt-4">
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Mail className="w-3 h-3" /> Recipient Emails
                            </label>
                            <div className="flex gap-2">
                                <Input
                                    type="text"
                                    placeholder="recipient@email.com"
                                    value={editRecipientInput}
                                    onChange={(e) => {
                                        setEditRecipientInput(e.target.value);
                                        if (error) setError(null);
                                    }}
                                    onKeyDown={handleEditRecipientKeyDown}
                                    className="bg-dark-950 border-dark-700 focus:border-teal-500 text-dark-100 placeholder:text-dark-500"
                                />
                                <Button
                                    type="button"
                                    variant="outline"
                                    className="border-dark-700 bg-dark-900 hover:bg-dark-800 text-dark-200"
                                    onClick={handleAddEditRecipient}
                                    disabled={!editRecipientInput.trim()}
                                >
                                    <Plus className="w-4 h-4 mr-1" /> Add
                                </Button>
                            </div>

                            {editRecipients.length > 0 && (
                                <div className="flex flex-wrap gap-2 bg-dark-900 border border-dark-700 rounded-lg p-2.5">
                                    {editRecipients.map((email) => (
                                        <div key={`edit-${email}`} className="flex items-center gap-1.5 bg-dark-800 text-dark-200 text-xs px-2 py-1 rounded max-w-full min-w-0">
                                            <Mail className="w-3 h-3 text-teal-400 shrink-0" />
                                            <span className="truncate max-w-[220px] sm:max-w-[300px]" title={email}>{email}</span>
                                            <button
                                                type="button"
                                                onClick={() => removeEditRecipient(email)}
                                                className="text-dark-400 hover:text-red-400 shrink-0"
                                            >
                                                <X className="w-3 h-3" />
                                            </button>
                                        </div>
                                    ))}
                                </div>
                            )}

                            <p className="text-[10px] text-dark-500">Press Enter, comma, semicolon, or use Add button.</p>
                        </div>

                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400">Message Content</label>
                            <Textarea
                                value={editContent}
                                onChange={(e) => setEditContent(e.target.value)}
                                className="min-h-[150px] bg-dark-950 border-dark-700 focus:border-teal-500 resize-none text-dark-100 placeholder:text-dark-500"
                                placeholder="Enter your message..."
                            />
                        </div>

                        {/* Attachments Toggle */}
                        <div className="flex items-center space-x-2 pt-2">
                            <input
                                type="checkbox"
                                id="edit-show-attachments"
                                checked={showEditAttachments}
                                onChange={(e) => {
                                    setShowEditAttachments(e.target.checked);
                                    if (!e.target.checked) setEditNewFiles([]);
                                }}
                                className="h-4 w-4 rounded border-dark-700 bg-dark-950 text-teal-600 focus:ring-teal-500 accent-teal-500"
                            />
                            <label htmlFor="edit-show-attachments" className="text-xs font-medium text-dark-300 cursor-pointer">
                                Send attachments with this switch
                            </label>
                        </div>

                        {/* Attachments Section */}
                        {showEditAttachments && (
                            <div className="space-y-2 animate-in fade-in slide-in-from-top-2 duration-300">
                                <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                    <Paperclip className="w-3 h-3" /> Attachments
                                    <span className="text-dark-600 font-normal">
                                        ({editAttachments.length + editNewFiles.length}/{MAX_FILES})
                                    </span>
                                </label>

                                {/* Existing attachments */}
                                {editAttachments.length > 0 && (
                                    <div className="space-y-1.5">
                                        {editAttachments.map(att => (
                                            <div
                                                key={att.id}
                                                className="flex items-center justify-between bg-dark-950 border border-dark-700 rounded-lg px-3 py-2"
                                            >
                                                <div className="flex items-center gap-2 min-w-0">
                                                    <Paperclip className="w-3 h-3 text-teal-400 shrink-0" />
                                                    <span className="text-xs text-dark-200 truncate">{att.filename}</span>
                                                    <span className="text-[10px] text-dark-500 shrink-0">{formatFileSize(att.size)}</span>
                                                </div>
                                                <button
                                                    onClick={() => handleDeleteAttachment(att.id)}
                                                    disabled={editAttachLoading}
                                                    className="text-dark-500 hover:text-red-400 transition-colors p-0.5"
                                                >
                                                    {editAttachLoading ? (
                                                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                                    ) : (
                                                        <X className="w-3.5 h-3.5" />
                                                    )}
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                )}

                                {/* New files to upload */}
                                {editNewFiles.length > 0 && (
                                    <div className="space-y-1.5">
                                        {editNewFiles.map((file, index) => (
                                            <div
                                                key={`new-${file.name}-${index}`}
                                                className="flex items-center justify-between bg-teal-500/5 border border-teal-500/20 rounded-lg px-3 py-2"
                                            >
                                                <div className="flex items-center gap-2 min-w-0">
                                                    <Upload className="w-3 h-3 text-teal-400 shrink-0" />
                                                    <span className="text-xs text-dark-200 truncate">{file.name}</span>
                                                    <span className="text-[10px] text-teal-400 shrink-0">new</span>
                                                    <span className="text-[10px] text-dark-500 shrink-0">{formatFileSize(file.size)}</span>
                                                </div>
                                                <button
                                                    onClick={() => setEditNewFiles(prev => prev.filter((_, i) => i !== index))}
                                                    className="text-dark-500 hover:text-red-400 transition-colors p-0.5"
                                                >
                                                    <X className="w-3.5 h-3.5" />
                                                </button>
                                            </div>
                                        ))}
                                    </div>
                                )}

                                {/* Add more files */}
                                {(editAttachments.length + editNewFiles.length) < MAX_FILES && (
                                    <button
                                        onClick={() => editFileInputRef.current?.click()}
                                        className="w-full border border-dashed border-dark-700 hover:border-dark-500 rounded-lg p-3 text-center transition-colors"
                                    >
                                        <input
                                            ref={editFileInputRef}
                                            type="file"
                                            multiple
                                            className="hidden"
                                            accept={ALLOWED_EXTENSIONS.join(',')}
                                            onChange={(e) => {
                                                if (e.target.files?.length) addEditFiles(e.target.files);
                                                e.target.value = '';
                                            }}
                                        />
                                        <Upload className="w-4 h-4 text-dark-500 mx-auto mb-1" />
                                        <p className="text-[10px] text-dark-500">Add files</p>
                                    </button>
                                )}
                            </div>
                        )}

                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Clock className="w-3 h-3" /> Trigger After
                            </label>
                            <Select
                                value={editDuration}
                                onChange={(e) => handleEditDurationChange(Number(e.target.value))}
                                className="bg-dark-950 border-dark-700 text-dark-100"
                            >
                                {TIME_PRESETS.map(preset => (
                                    <option key={preset.value} value={preset.value}>
                                        {preset.label}
                                    </option>
                                ))}
                            </Select>
                            <p className="text-[10px] text-dark-500">
                                Timer will reset when you save changes
                            </p>
                        </div>

                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Clock className="w-3 h-3 text-teal-400" /> Reminders Before Trigger
                            </label>
                            <div className="flex flex-col gap-2 bg-dark-900 border border-dark-700 rounded-lg p-3">
                                {editReminders.length > 0 ? (
                                    <div className="flex flex-wrap gap-2">
                                        {editReminders.map(r => {
                                            const preset = REMINDER_PRESETS.find(p => p.value === r);
                                            const label = preset ? preset.label : formatMinutes(r);
                                            return (
                                                <div key={r} className="flex items-center gap-1 bg-dark-800 text-dark-200 text-xs px-2 py-1 rounded">
                                                    <span>{label}</span>
                                                    <button type="button" onClick={() => removeEditReminder(r)} className="text-dark-400 hover:text-red-400">
                                                        <X className="w-3 h-3" />
                                                    </button>
                                                </div>
                                            );
                                        })}
                                    </div>
                                ) : (
                                    <p className="text-xs text-dark-500">No reminders configured. The switch will trigger without warning.</p>
                                )}

                                <div className="flex items-center gap-2 mt-2">
                                    <Select
                                        onChange={(e) => {
                                            if (e.target.value) {
                                                addEditReminder(Number(e.target.value));
                                                e.target.value = '';
                                            }
                                        }}
                                        className="bg-dark-950 border-dark-700 text-dark-100 text-xs h-8"
                                        value={""}
                                    >
                                        <option value="" disabled>Add a reminder...</option>
                                        {REMINDER_PRESETS.filter(p => !editReminders.includes(p.value) && p.value < editDuration).map(preset => (
                                            <option key={preset.value} value={preset.value}>
                                                {preset.label}
                                            </option>
                                        ))}
                                    </Select>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="border-t border-dark-700 pt-4 mt-2">
                        <FarewellLetters messageId={editingMessage?.id} onChanged={fetchMessages} />
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
                            disabled={actionLoading || !editContent.trim() || (editRecipients.length === 0 && !editRecipientInput.trim())}
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

            <Dialog open={queueDialogOpen} onOpenChange={(open) => {
                setQueueDialogOpen(open);
                if (!open) setQueueMessage(null);
            }}>
                <DialogContent className="max-w-[95vw] sm:max-w-2xl max-h-[90vh] overflow-y-auto bg-dark-900 border-dark-700">
                    <DialogHeader>
                        <DialogTitle>Manage Queue</DialogTitle>
                        <DialogDescription className="text-dark-400">
                            {queueMessage
                                ? `Triggered switch for ${formatRecipientEmails(queueMessage.recipient_email)}`
                                : 'Triggered switch'}
                        </DialogDescription>
                    </DialogHeader>
                    <div className="mt-4">
                        <FarewellLetters
                            messageId={queueMessage?.id}
                            mode="queue"
                            onChanged={fetchMessages}
                        />
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
}
