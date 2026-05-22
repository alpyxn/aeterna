import { useState, useEffect, useMemo, useRef, useCallback } from 'react';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogTitle, AlertDialogDescription, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog";
import {
    Mail, Clock, Trash2, Paperclip, X, Plus, Loader2, CheckCircle,
    AlertCircle, Eye, Pencil, Upload, Send
} from 'lucide-react';
import {
    listFarewellLetters, createFarewellLetter, updateFarewellLetter,
    deleteFarewellLetter, cancelFarewellLetter, cancelAllPendingFarewellLetters, uploadFarewellAttachment,
    listFarewellAttachments, deleteFarewellAttachment
} from "@/lib/api";
import { ALLOWED_EXTENSIONS, MAX_FILE_SIZE, MAX_FILES, MAX_TOTAL_SIZE, FAREWELL_DELAY_PRESETS, EMAIL_REGEX } from "@/lib/constants"
import { formatFileSize, formatFarewellDelay } from "@/lib/formatters"

function renderSafeMarkdown(markdown) {
    const rendered = marked.parse(markdown || '');
    const html = typeof rendered === 'string' ? rendered : '';
    return DOMPurify.sanitize(html, {
        USE_PROFILES: { html: true },
    });
}

function MarkdownEditor({ value, onChange }) {
    const [preview, setPreview] = useState(false);
    const previewHtml = useMemo(() => renderSafeMarkdown(value), [value]);

    return (
        <div className="space-y-1">
            <div className="flex items-center justify-between">
                <span className="text-xs text-dark-400">Supports Markdown formatting</span>
                <button
                    type="button"
                    onClick={() => setPreview(!preview)}
                    className="flex items-center gap-1 text-xs text-teal-400 hover:text-teal-300"
                >
                    {preview ? <><Pencil className="w-3 h-3" /> Edit</> : <><Eye className="w-3 h-3" /> Preview</>}
                </button>
            </div>
            {preview ? (
                <div
                    className="min-h-[160px] p-3 rounded-md border border-dark-700 bg-dark-950 text-dark-100 text-sm prose prose-invert max-w-none overflow-auto"
                    dangerouslySetInnerHTML={{ __html: previewHtml }}
                />
            ) : (
                <Textarea
                    value={value}
                    onChange={(e) => onChange(e.target.value)}
                    placeholder="Write your farewell message here... Markdown is supported."
                    className="min-h-[160px] bg-dark-950 border-dark-700 focus:border-teal-500 text-dark-100 placeholder:text-dark-500 font-mono text-sm"
                />
            )}
        </div>
    );
}

function AttachmentManager({ messageId, letterId: propLetterId, disabled, pendingFiles = [], onPendingFilesChange }) {
    const [letterIdState, setLetterIdState] = useState(propLetterId);
    const [attachments, setAttachments] = useState([]);
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState(null);

    const loadAttachments = useCallback(async (letterId) => {
        if (!letterId) return;
        try {
            const data = await listFarewellAttachments(messageId, letterId);
            setAttachments(data || []);
        } catch {
            // non-fatal
        }
    }, [messageId]);

    useEffect(() => {
        setLetterIdState(propLetterId);
        if (propLetterId) {
            loadAttachments(propLetterId);
        } else {
            setAttachments([]);
        }
    }, [propLetterId, loadAttachments]);

    function validateFile(file) {
        const ext = '.' + file.name.split('.').pop().toLowerCase();
        if (!ALLOWED_EXTENSIONS.includes(ext)) return `File type ${ext} is not allowed`;
        if (file.size > MAX_FILE_SIZE) return `File exceeds 10 MB limit`;
        if (attachments.length + pendingFiles.length >= MAX_FILES) return `Maximum ${MAX_FILES} attachments`;
        const totalSize = attachments.reduce((s, a) => s + a.size, 0) + pendingFiles.reduce((s, f) => s + f.size, 0);
        if (totalSize + file.size > MAX_TOTAL_SIZE) return `Total size would exceed 25 MB`;
        return null;
    }

    async function handleUpload(e) {
        const file = e.target.files?.[0];
        if (!file) return;
        const err = validateFile(file);
        if (err) { setError(err); e.target.value = ''; return; }

        const lid = letterIdState;

        if (!lid) {
            onPendingFilesChange?.(prev => [...prev, file]);
            setError(null);
            e.target.value = '';
            return;
        }

        setUploading(true);
        setError(null);
        try {
            await uploadFarewellAttachment(messageId, lid, file);
            await loadAttachments(lid);
        } catch (err) {
            setError(err.message);
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    }

    function removePendingFile(indexToRemove) {
        onPendingFilesChange?.(prev => prev.filter((_, index) => index !== indexToRemove));
    }

    async function handleDelete(attachmentId) {
        try {
            await deleteFarewellAttachment(messageId, letterIdState, attachmentId);
            setAttachments(prev => prev.filter(a => a.id !== attachmentId));
        } catch (err) {
            setError(err.message);
        }
    }

    return (
        <div className="space-y-2">
            <div className="flex items-center justify-between">
                <label className="text-xs font-medium text-dark-400 flex items-center gap-1">
                    <Paperclip className="w-3 h-3" /> Attachments ({attachments.length + pendingFiles.length}/{MAX_FILES})
                </label>
                {!disabled && (
                    <label className="cursor-pointer flex items-center gap-1 text-xs text-teal-400 hover:text-teal-300">
                        <Upload className="w-3 h-3" />
                        {uploading ? "Uploading..." : (letterIdState ? "Add file" : "Queue file")}
                        <input type="file" className="hidden" onChange={handleUpload} disabled={uploading || attachments.length + pendingFiles.length >= MAX_FILES} accept={ALLOWED_EXTENSIONS.join(',')} />
                    </label>
                )}
            </div>
            {error && <p className="text-xs text-red-400">{error}</p>}
            {!letterIdState && pendingFiles.length > 0 && (
                <p className="text-xs text-dark-500">Files will upload when you save the letter.</p>
            )}
            {attachments.length > 0 && (
                <div className="space-y-1">
                    {attachments.map(att => (
                        <div key={att.id} className="flex items-center justify-between bg-dark-900 border border-dark-700 rounded px-2 py-1.5">
                            <div className="flex items-center gap-2 min-w-0">
                                <Paperclip className="w-3 h-3 text-dark-400 shrink-0" />
                                <span className="text-xs text-dark-200 truncate">{att.filename}</span>
                                <span className="text-xs text-dark-500 shrink-0">{formatFileSize(att.size)}</span>
                            </div>
                            {!disabled && (
                                <button type="button" onClick={() => handleDelete(att.id)} className="text-dark-500 hover:text-red-400 ml-2">
                                    <X className="w-3 h-3" />
                                </button>
                            )}
                        </div>
                    ))}
                </div>
            )}
            {pendingFiles.length > 0 && (
                <div className="space-y-1">
                    {pendingFiles.map((file, index) => (
                        <div key={`${file.name}-${file.size}-${index}`} className="flex items-center justify-between bg-dark-900 border border-dark-700 rounded px-2 py-1.5">
                            <div className="flex items-center gap-2 min-w-0">
                                <Paperclip className="w-3 h-3 text-dark-400 shrink-0" />
                                <span className="text-xs text-dark-200 truncate">{file.name}</span>
                                <span className="text-xs text-dark-500 shrink-0">{formatFileSize(file.size)}</span>
                                <span className="text-[10px] text-amber-400 shrink-0">Queued</span>
                            </div>
                            {!disabled && (
                                <button type="button" onClick={() => removePendingFile(index)} className="text-dark-500 hover:text-red-400 ml-2">
                                    <X className="w-3 h-3" />
                                </button>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}

function LetterForm({ messageId, letter, onSave, onCancel }) {
    const [recipientEmail, setRecipientEmail] = useState(letter?.recipient_email || '');
    const [subject, setSubject] = useState(letter?.subject || '');
    const [content, setContent] = useState(letter?.content || '');
    const [delayMinutes, setDelayMinutes] = useState(letter?.delay_minutes ?? 1440);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState(null);
    const [savedLetterId, setSavedLetterId] = useState(letter?.id || null);
    const [pendingFiles, setPendingFiles] = useState([]);

    async function handleSave() {
        setError(null);

        if (!recipientEmail.trim() || !EMAIL_REGEX.test(recipientEmail.trim())) {
            setError('Please enter a valid recipient email address.');
            return;
        }
        if (!subject.trim()) {
            setError('Subject is required.');
            return;
        }
        if (!content.trim()) {
            setError('Message content is required.');
            return;
        }

        setSaving(true);
        try {
            const isNewLetter = !savedLetterId;
            const payload = { recipient_email: recipientEmail, subject, content, delay_minutes: delayMinutes };
            let saved;
            if (savedLetterId) {
                saved = await updateFarewellLetter(messageId, savedLetterId, payload);
            } else {
                saved = await createFarewellLetter(messageId, payload);
            }

            for (const file of pendingFiles) {
                await uploadFarewellAttachment(messageId, saved.id, file);
                setPendingFiles(prev => prev.filter(f => f !== file));
            }

            setSavedLetterId(saved.id);
            onSave(saved, { isNewLetter });
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    }

    const isSent = letter?.status === 'sent';

    return (
        <div className="space-y-4 p-4 border border-dark-700 rounded-lg bg-dark-900">
            {error && (
                <Alert variant="destructive" className="border-red-800 bg-red-950/50">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div className="space-y-1">
                    <label className="text-xs font-medium text-dark-400 flex items-center gap-1">
                        <Mail className="w-3 h-3" /> Recipient email
                    </label>
                    <Input
                        type="email"
                        placeholder="someone@example.com"
                        value={recipientEmail}
                        onChange={(e) => setRecipientEmail(e.target.value)}
                        disabled={isSent}
                        className="bg-dark-950 border-dark-700 focus:border-teal-500 text-dark-100 placeholder:text-dark-500"
                    />
                </div>
                <div className="space-y-1">
                    <label className="text-xs font-medium text-dark-400 flex items-center gap-1">
                        <Clock className="w-3 h-3" /> Send delay
                    </label>
                    <select
                        value={delayMinutes}
                        onChange={(e) => setDelayMinutes(Number(e.target.value))}
                        disabled={isSent}
                        className="w-full h-9 rounded-md border border-dark-700 bg-dark-950 text-dark-100 text-sm px-3 focus:outline-none focus:border-teal-500"
                    >
                        {FAREWELL_DELAY_PRESETS.map(p => (
                            <option key={p.value} value={p.value}>{p.label}</option>
                        ))}
                    </select>
                </div>
            </div>

            <div className="space-y-1">
                <label className="text-xs font-medium text-dark-400">Subject</label>
                <Input
                    type="text"
                    placeholder="A farewell message for you"
                    value={subject}
                    onChange={(e) => setSubject(e.target.value)}
                    disabled={isSent}
                    className="bg-dark-950 border-dark-700 focus:border-teal-500 text-dark-100 placeholder:text-dark-500"
                />
            </div>

            <div className="space-y-1">
                <label className="text-xs font-medium text-dark-400">Message</label>
                <MarkdownEditor value={content} onChange={setContent} />
            </div>

            <AttachmentManager
                messageId={messageId}
                letterId={savedLetterId}
                disabled={isSent || saving}
                pendingFiles={pendingFiles}
                onPendingFilesChange={setPendingFiles}
            />

            {isSent ? (
                <p className="text-xs text-teal-400 flex items-center gap-1">
                    <Send className="w-3 h-3" /> This letter has already been sent
                </p>
            ) : (
                <div className="flex gap-2 justify-end">
                    <Button variant="outline" size="sm" onClick={onCancel}
                        className="border-dark-700 bg-dark-900 hover:bg-dark-800 text-dark-200">
                        {savedLetterId ? 'Done' : 'Cancel'}
                    </Button>
                    <Button size="sm" onClick={handleSave} disabled={saving}
                        className="bg-teal-600 hover:bg-teal-500 text-white">
                        {saving
                            ? <><Loader2 className="w-3 h-3 mr-1 animate-spin" /> Saving...</>
                            : (savedLetterId ? 'Save changes' : 'Save letter')}
                    </Button>
                </div>
            )}
        </div>
    );
}

export default function FarewellLetters({ messageId, mode = 'edit', onChanged }) {
    const [letters, setLetters] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [editingLetter, setEditingLetter] = useState(null);
    const [showForm, setShowForm] = useState(false);
    const [toastMessage, setToastMessage] = useState('');
    const toastTimeoutRef = useRef(null);
    const queueMode = mode === 'queue';
    const pendingLetters = letters.filter(letter => letter.status === 'pending');

    const loadLetters = useCallback(async () => {
        if (!messageId) return;
        setLoading(true);
        try {
            const data = await listFarewellLetters(messageId);
            setLetters(data || []);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, [messageId]);

    useEffect(() => {
        loadLetters();
    }, [loadLetters]);

    useEffect(() => {
        return () => {
            if (toastTimeoutRef.current) {
                clearTimeout(toastTimeoutRef.current);
            }
        };
    }, []);

    function showToast(message) {
        setToastMessage(message);
        if (toastTimeoutRef.current) {
            clearTimeout(toastTimeoutRef.current);
        }
        toastTimeoutRef.current = setTimeout(() => {
            setToastMessage('');
            toastTimeoutRef.current = null;
        }, 3000);
    }

    function handleSaved(saved, meta = {}) {
        setLetters(prev => {
            const idx = prev.findIndex(l => l.id === saved.id);
            if (idx >= 0) {
                const next = [...prev];
                next[idx] = saved;
                return next;
            }
            return [...prev, saved];
        });
        setShowForm(false);
        setEditingLetter(null);
        onChanged?.();
        showToast(meta.isNewLetter ? 'Letter saved to switch' : 'Letter changes saved');
    }

    function handleDismiss() {
        setShowForm(false);
        setEditingLetter(null);
    }

    async function handleDelete(letterId) {
        try {
            await deleteFarewellLetter(messageId, letterId);
            setLetters(prev => prev.filter(l => l.id !== letterId));
            onChanged?.();
        } catch (err) {
            setError(err.message);
        }
    }

    async function handleCancel(letterId) {
        try {
            await cancelFarewellLetter(messageId, letterId);
            setLetters(prev => prev.filter(l => l.id !== letterId));
            onChanged?.();
            showToast('Pending delivery canceled');
        } catch (err) {
            setError(err.message);
        }
    }

    async function handleCancelAll() {
        try {
            const result = await cancelAllPendingFarewellLetters(messageId);
            setLetters(prev => prev.filter(l => l.status !== 'pending'));
            onChanged?.();
            showToast(`${result?.canceled || 0} pending deliver${result?.canceled === 1 ? 'y' : 'ies'} canceled`);
        } catch (err) {
            setError(err.message);
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center py-6 text-dark-400">
                <Loader2 className="w-4 h-4 animate-spin mr-2" /> Loading farewell letters...
            </div>
        );
    }

    return (
        <div className="space-y-3">
            {toastMessage && (
                <div className="fixed right-4 top-20 z-[300] max-w-xs rounded-lg border border-teal-500/30 bg-dark-900 px-3 py-2 shadow-lg">
                    <div className="flex items-center gap-2 text-sm text-teal-300">
                        <CheckCircle className="h-4 w-4 shrink-0 text-teal-400" />
                        <span>{toastMessage}</span>
                    </div>
                </div>
            )}
            <div className="flex items-center justify-between">
                <div>
                    <h3 className="text-sm font-medium text-dark-100">Farewell Letters</h3>
                    <p className="text-xs text-dark-400 mt-0.5">
                        {queueMode
                            ? 'Pending deliveries can be canceled after this switch has triggered.'
                            : 'Personal messages sent after this switch fires, each with its own delay.'}
                    </p>
                </div>
                {queueMode && pendingLetters.length > 0 && (
                    <AlertDialog>
                        <AlertDialogTrigger asChild>
                            <Button size="sm" variant="outline"
                                className="border-red-500/30 bg-red-950/20 hover:bg-red-950/40 text-red-300 shrink-0">
                                <Trash2 className="w-3 h-3 mr-1" /> Cancel all
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent className="bg-dark-900 border-dark-700">
                            <AlertDialogTitle>Cancel all pending letters?</AlertDialogTitle>
                            <AlertDialogDescription className="text-dark-400">
                                This cancels every pending farewell letter for this triggered switch. Sent letters are not changed.
                            </AlertDialogDescription>
                            <div className="flex gap-2 justify-end mt-2">
                                <AlertDialogCancel className="border-dark-700 bg-dark-900 hover:bg-dark-800 text-dark-200">
                                    Keep queue
                                </AlertDialogCancel>
                                <AlertDialogAction onClick={handleCancelAll}
                                    className="bg-red-600 hover:bg-red-500 text-white border-0">
                                    Cancel all pending
                                </AlertDialogAction>
                            </div>
                        </AlertDialogContent>
                    </AlertDialog>
                )}
                {!queueMode && !showForm && !editingLetter && (
                    <Button size="sm" variant="outline" onClick={() => setShowForm(true)}
                        className="border-dark-700 bg-dark-900 hover:bg-dark-800 text-dark-200 shrink-0">
                        <Plus className="w-3 h-3 mr-1" /> Add letter
                    </Button>
                )}
            </div>

            {error && (
                <Alert variant="destructive" className="border-red-800 bg-red-950/50">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            {(showForm && !editingLetter) && (
                <LetterForm
                    messageId={messageId}
                    onSave={handleSaved}
                    onCancel={handleDismiss}
                />
            )}

            {queueMode && letters.length > 0 && pendingLetters.length === 0 && (
                <Alert className="border-teal-500/20 bg-teal-500/10">
                    <CheckCircle className="h-4 w-4 text-teal-400" />
                    <AlertDescription className="text-teal-200">
                        No pending farewell deliveries remain for this switch.
                    </AlertDescription>
                </Alert>
            )}

            {letters.length === 0 && !showForm ? (
                <div className="text-center py-6 text-dark-500 text-xs border border-dashed border-dark-700 rounded-lg">
                    {queueMode ? 'No farewell letters are queued for this switch.' : 'No farewell letters configured yet.'}
                </div>
            ) : (
                <div className="space-y-2">
                    {letters.map(letter => (
                        <div key={letter.id}>
                            {editingLetter?.id === letter.id ? (
                                <LetterForm
                                    messageId={messageId}
                                    letter={letter}
                                    onSave={handleSaved}
                                    onCancel={handleDismiss}
                                />
                            ) : (
                                <div className="flex items-start justify-between gap-3 p-3 border border-dark-700 rounded-lg bg-dark-900 hover:border-dark-600 transition-colors">
                                    <div className="min-w-0 flex-1 space-y-1">
                                        <div className="flex items-center gap-2 flex-wrap">
                                            <span className="text-xs font-medium text-dark-100 truncate">{letter.subject}</span>
                                            {letter.status === 'sent' ? (
                                                <span className="text-xs px-1.5 py-0.5 rounded bg-teal-900/50 text-teal-400 border border-teal-800 shrink-0">Sent</span>
                                            ) : (
                                                <span className="text-xs px-1.5 py-0.5 rounded bg-dark-800 text-dark-400 border border-dark-700 shrink-0">Pending</span>
                                            )}
                                        </div>
                                        <div className="flex items-center gap-3 text-xs text-dark-400">
                                            <span className="flex items-center gap-1"><Mail className="w-3 h-3" />{letter.recipient_email}</span>
                                            <span className="flex items-center gap-1"><Clock className="w-3 h-3" />{formatFarewellDelay(letter.delay_minutes)}</span>
                                            {letter.attachment_count > 0 && (
                                                <span className="flex items-center gap-1"><Paperclip className="w-3 h-3" />{letter.attachment_count}</span>
                                            )}
                                        </div>
                                    </div>
                                    {letter.status !== 'sent' && !queueMode && (
                                        <div className="flex items-center gap-1 shrink-0">
                                            <Button size="sm" variant="ghost" onClick={() => setEditingLetter(letter)}
                                                className="h-7 w-7 p-0 text-dark-400 hover:text-dark-100 hover:bg-dark-800">
                                                <Pencil className="w-3 h-3" />
                                            </Button>
                                            <AlertDialog>
                                                <AlertDialogTrigger asChild>
                                                    <Button size="sm" variant="ghost"
                                                        className="h-7 w-7 p-0 text-dark-400 hover:text-red-400 hover:bg-red-950/30">
                                                        <Trash2 className="w-3 h-3" />
                                                    </Button>
                                                </AlertDialogTrigger>
                                                <AlertDialogContent className="bg-dark-900 border-dark-700">
                                                    <AlertDialogTitle>Delete farewell letter?</AlertDialogTitle>
                                                    <AlertDialogDescription className="text-dark-400">
                                                        This letter and its attachments will be permanently deleted.
                                                    </AlertDialogDescription>
                                                    <div className="flex gap-2 justify-end mt-2">
                                                        <AlertDialogCancel className="border-dark-700 bg-dark-900 hover:bg-dark-800 text-dark-200">
                                                            Cancel
                                                        </AlertDialogCancel>
                                                        <AlertDialogAction onClick={() => handleDelete(letter.id)}
                                                            className="bg-red-600 hover:bg-red-500 text-white border-0">
                                                            Delete
                                                        </AlertDialogAction>
                                                    </div>
                                                </AlertDialogContent>
                                            </AlertDialog>
                                        </div>
                                    )}
                                    {letter.status !== 'sent' && queueMode && (
                                        <AlertDialog>
                                            <AlertDialogTrigger asChild>
                                                <Button size="sm" variant="outline"
                                                    className="h-8 border-red-500/30 bg-red-950/20 hover:bg-red-950/40 text-red-300 shrink-0">
                                                    <Trash2 className="w-3 h-3 mr-1" /> Cancel
                                                </Button>
                                            </AlertDialogTrigger>
                                            <AlertDialogContent className="bg-dark-900 border-dark-700">
                                                <AlertDialogTitle>Cancel pending letter?</AlertDialogTitle>
                                                <AlertDialogDescription className="text-dark-400">
                                                    This prevents this farewell letter from being delivered. Sent letters cannot be canceled.
                                                </AlertDialogDescription>
                                                <div className="flex gap-2 justify-end mt-2">
                                                    <AlertDialogCancel className="border-dark-700 bg-dark-900 hover:bg-dark-800 text-dark-200">
                                                        Keep letter
                                                    </AlertDialogCancel>
                                                    <AlertDialogAction onClick={() => handleCancel(letter.id)}
                                                        className="bg-red-600 hover:bg-red-500 text-white border-0">
                                                        Cancel delivery
                                                    </AlertDialogAction>
                                                </div>
                                            </AlertDialogContent>
                                        </AlertDialog>
                                    )}
                                </div>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
