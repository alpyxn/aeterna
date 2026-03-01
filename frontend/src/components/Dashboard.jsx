import { useState, useEffect, useCallback, useRef } from 'react';
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogTitle, AlertDialogDescription, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog"
import { Mail, Clock, Loader2, Trash2, Heart, AlertCircle, RefreshCw, Inbox, Eye, Pencil, Paperclip, X, Upload } from 'lucide-react';
import { apiRequest, uploadFile, deleteAttachment, listAttachments } from "@/lib/api";
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { Select } from "@/components/ui/select"

const ALLOWED_EXTENSIONS = ['.pdf', '.txt', '.doc', '.docx', '.jpg', '.jpeg', '.png', '.gif', '.webp', '.zip'];
const MAX_FILE_SIZE = 10 * 1024 * 1024;
const MAX_FILES = 5;
const MAX_TOTAL_SIZE = 25 * 1024 * 1024;

function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

function formatMinutes(minutes) {
    if (minutes >= 1440) {
        // e.g., 2880 -> 2, 3600 -> 2.5
        const days = Number((minutes / 1440).toFixed(1));
        return `${days} Day${days !== 1 ? 's' : ''} Before`;
    }
    if (minutes >= 60) {
        const hours = Number((minutes / 60).toFixed(1));
        return `${hours} Hour${hours !== 1 ? 's' : ''} Before`;
    }
    return `${minutes} Minutes Before`;
}

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
    const [editReminders, setEditReminders] = useState([]);
    const [editDialogOpen, setEditDialogOpen] = useState(false);
    const [editAttachments, setEditAttachments] = useState([]);
    const [editNewFiles, setEditNewFiles] = useState([]);
    const [showEditAttachments, setShowEditAttachments] = useState(false);
    const [editAttachLoading, setEditAttachLoading] = useState(false);
    const editFileInputRef = useRef(null);

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

    const reminderPresets = [
        { label: '15 Minutes Before', value: 15 },
        { label: '1 Hour Before', value: 60 },
        { label: '12 Hours Before', value: 720 },
        { label: '1 Day Before', value: 1440 },
        { label: '2 Days Before', value: 2880 },
        { label: '3 Days Before', value: 4320 },
        { label: '5 Days Before', value: 7200 },
        { label: '10 Days Before', value: 14400 },
    ];

    const handleEditDurationChange = (newDuration) => {
        setEditDuration(newDuration);
        const validReminders = editReminders.filter(r => r < newDuration);
        if (validReminders.length === 0) {
            if (newDuration >= 1440) {
                setEditReminders([newDuration / 2]);
            } else if (newDuration >= 60) {
                setEditReminders([15]);
            } else {
                setEditReminders([]);
            }
        } else {
            setEditReminders(validReminders);
        }
    };

    const addEditReminder = (value) => {
        if (!editReminders.includes(value) && value < editDuration) {
            setEditReminders([...editReminders, value].sort((a, b) => b - a)); // sort descending
        }
    };

    const removeEditReminder = (value) => {
        setEditReminders(editReminders.filter(r => r !== value));
    };

    const openEditDialog = async (message) => {
        setEditingMessage(message);
        setEditContent(message.content);
        setEditDuration(message.trigger_duration);
        setEditReminders(message.reminders ? message.reminders.map(r => r.minutes_before) : []);
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

        setActionLoading(editingMessage.id);
        try {
            await apiRequest(`/messages/${editingMessage.id}`, {
                method: 'PUT',
                body: JSON.stringify({
                    content: editContent,
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

                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Clock className="w-3 h-3 text-teal-400" /> Reminders Before Trigger
                            </label>
                            <div className="flex flex-col gap-2 bg-dark-900 border border-dark-700 rounded-lg p-3">
                                {editReminders.length > 0 ? (
                                    <div className="flex flex-wrap gap-2">
                                        {editReminders.map(r => {
                                            const preset = reminderPresets.find(p => p.value === r);
                                            const label = preset ? preset.label : formatMinutes(r);
                                            return (
                                                <div key={r} className="flex items-center gap-1 bg-dark-800 text-dark-200 text-xs px-2 py-1 rounded">
                                                    <span>{label}</span>
                                                    <button onClick={() => removeEditReminder(r)} className="text-dark-400 hover:text-red-400">
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
                                        {reminderPresets.filter(p => !editReminders.includes(p.value) && p.value < editDuration).map(preset => (
                                            <option key={preset.value} value={preset.value}>
                                                {preset.label}
                                            </option>
                                        ))}
                                    </Select>
                                </div>
                            </div>
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
