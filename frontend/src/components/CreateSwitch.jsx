import { useState, useRef } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Lock, Mail, Clock, Loader2, AlertCircle, CheckCircle, Send, Paperclip, X, Upload, Settings as SettingsIcon } from 'lucide-react';
import { Select } from "@/components/ui/select"
import { apiRequest, uploadFile } from "@/lib/api"

const ALLOWED_EXTENSIONS = ['.pdf', '.txt', '.doc', '.docx', '.jpg', '.jpeg', '.png', '.gif', '.webp', '.zip'];
const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10 MB
const MAX_FILES = 5;
const MAX_TOTAL_SIZE = 25 * 1024 * 1024; // 25 MB

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

export default function CreateSwitch({ setRoute }) {
    const [message, setMessage] = useState('');
    const [email, setEmail] = useState('');
    const [duration, setDuration] = useState(1440);
    const [reminders, setReminders] = useState([720]); // default to 12 hours before trigger
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);
    const [success, setSuccess] = useState(false);
    const [files, setFiles] = useState([]);
    const [uploadProgress, setUploadProgress] = useState('');
    const [showAttachments, setShowAttachments] = useState(false);
    const [dragOver, setDragOver] = useState(false);
    const [smtpError, setSmtpError] = useState(false);
    const fileInputRef = useRef(null);

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

    const handleDurationChange = (newDuration) => {
        setDuration(newDuration);
        // Add sensible default reminders if none are valid for the new duration
        const validReminders = reminders.filter(r => r < newDuration);
        if (validReminders.length === 0) {
            if (newDuration >= 1440) { // >= 1 day
                setReminders([newDuration / 2]); // 50%
            } else if (newDuration >= 60) { // >= 1 hour
                setReminders([15]); // 15 mins
            } else {
                setReminders([]);
            }
        } else {
            setReminders(validReminders);
        }
    };

    const addReminder = (value) => {
        if (!reminders.includes(value) && value < duration) {
            setReminders([...reminders, value].sort((a, b) => b - a)); // sort descending (longest before trigger first)
        }
    };

    const removeReminder = (value) => {
        setReminders(reminders.filter(r => r !== value));
    };

    const validateFile = (file) => {
        const ext = '.' + file.name.split('.').pop().toLowerCase();
        if (!ALLOWED_EXTENSIONS.includes(ext)) {
            return `"${file.name}" — type not allowed. Allowed: PDF, TXT, DOC, DOCX, JPG, PNG, GIF, WEBP, ZIP`;
        }
        if (file.size > MAX_FILE_SIZE) {
            return `"${file.name}" exceeds 10 MB limit`;
        }
        return null;
    };

    const addFiles = (newFiles) => {
        setError(null);
        const fileArray = Array.from(newFiles);

        if (files.length + fileArray.length > MAX_FILES) {
            setError(`Maximum ${MAX_FILES} files allowed`);
            return;
        }

        const currentTotal = files.reduce((sum, f) => sum + f.size, 0);
        const newTotal = fileArray.reduce((sum, f) => sum + f.size, 0);
        if (currentTotal + newTotal > MAX_TOTAL_SIZE) {
            setError('Total attachment size exceeds 25 MB limit');
            return;
        }

        for (const file of fileArray) {
            const err = validateFile(file);
            if (err) {
                setError(err);
                return;
            }
        }

        // Deduplicate by name
        const existingNames = new Set(files.map(f => f.name));
        const uniqueNew = fileArray.filter(f => !existingNames.has(f.name));
        setFiles(prev => [...prev, ...uniqueNew]);
    };

    const removeFile = (index) => {
        setFiles(prev => prev.filter((_, i) => i !== index));
    };

    const handleDrop = (e) => {
        e.preventDefault();
        setDragOver(false);
        if (e.dataTransfer.files?.length) {
            addFiles(e.dataTransfer.files);
        }
    };

    const handleDragOver = (e) => {
        e.preventDefault();
        setDragOver(true);
    };

    const handleDragLeave = (e) => {
        e.preventDefault();
        setDragOver(false);
    };

    const handleCreate = async () => {
        if (!message.trim()) {
            setError('Please enter a message');
            return;
        }
        if (!email.trim()) {
            setError('Please enter recipient email');
            return;
        }
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
            setError('Please enter a valid email address');
            return;
        }

        setLoading(true);
        setError(null);
        setSuccess(false);
        setUploadProgress('');

        try {
            // Step 1: Create the message
            const result = await apiRequest('/messages', {
                method: 'POST',
                body: JSON.stringify({
                    content: message,
                    recipient_email: email,
                    trigger_duration: duration,
                    reminders: reminders
                })
            }).catch(err => {
                if (err.message.includes('SMTP_NOT_CONFIGURED') || err.message.includes('SMTP_CONNECTION_FAILED')) {
                    setSmtpError(true);
                }
                throw err;
            });

            // Step 2: Upload files (if any)
            if (files.length > 0) {
                for (let i = 0; i < files.length; i++) {
                    setUploadProgress(`Uploading file ${i + 1}/${files.length}...`);
                    try {
                        await uploadFile(result.id, files[i]);
                    } catch (uploadErr) {
                        setError(`Switch created, but file "${files[i].name}" failed: ${uploadErr.message}`);
                        setLoading(false);
                        setUploadProgress('');
                        setFiles([]);
                        setMessage('');
                        setEmail('');
                        return;
                    }
                }
            }

            setSuccess(true);
            setMessage('');
            setEmail('');
            setFiles([]);
            setUploadProgress('');

            setTimeout(() => setSuccess(false), 5000);
        } catch (e) {
            setError(e.message);
        } finally {
            setLoading(false);
            setUploadProgress('');
        }
    };

    return (
        <div className="w-full max-w-2xl space-y-6">
            <div className="text-center space-y-2">
                <h1 className="text-2xl font-semibold text-dark-100">
                    Dead Man's Switch
                </h1>
                <p className="text-dark-400 text-sm max-w-md mx-auto">
                    Create a message that will be delivered if you don't check in regularly
                </p>
            </div>

            <Card className="glowing-card">
                <CardHeader className="pb-4">
                    <CardTitle className="flex items-center gap-2 text-base font-medium">
                        <Send className="w-4 h-4 text-teal-400" />
                        Create New Switch
                    </CardTitle>
                    <CardDescription className="text-dark-400">
                        Your message will be sent if you fail to send a heartbeat before the timer runs out
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="space-y-2">
                        <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                            <Lock className="w-3 h-3" /> Your Message
                        </label>
                        <Textarea
                            placeholder="Write your message here..."
                            value={message}
                            onChange={(e) => {
                                setMessage(e.target.value);
                                if (error) setError(null);
                                if (success) setSuccess(false);
                            }}
                            className="min-h-[120px] bg-dark-950 border-dark-700 focus:border-teal-500 resize-none text-dark-100 placeholder:text-dark-500"
                        />
                    </div>

                    {/* Attachments Toggle */}
                    <div className="flex items-center space-x-2 pt-2">
                        <input
                            type="checkbox"
                            id="show-attachments"
                            checked={showAttachments}
                            onChange={(e) => {
                                setShowAttachments(e.target.checked);
                                if (!e.target.checked) setFiles([]);
                            }}
                            className="h-4 w-4 rounded border-dark-700 bg-dark-950 text-teal-600 focus:ring-teal-500 accent-teal-500"
                        />
                        <label htmlFor="show-attachments" className="text-xs font-medium text-dark-300 cursor-pointer">
                            Send attachments with this switch
                        </label>
                    </div>

                    {/* File Upload Area */}
                    {showAttachments && (
                        <div className="space-y-2 animate-in fade-in slide-in-from-top-2 duration-300">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Paperclip className="w-3 h-3" /> Attachments
                                <span className="text-dark-600 font-normal">({files.length}/{MAX_FILES})</span>
                            </label>
                            <div
                                className={`border-2 border-dashed rounded-lg p-4 text-center cursor-pointer transition-all ${dragOver
                                    ? 'border-teal-400 bg-teal-500/5'
                                    : 'border-dark-700 hover:border-dark-500 bg-dark-950'
                                    }`}
                                onDrop={handleDrop}
                                onDragOver={handleDragOver}
                                onDragLeave={handleDragLeave}
                                onClick={() => fileInputRef.current?.click()}
                            >
                                <input
                                    ref={fileInputRef}
                                    type="file"
                                    multiple
                                    className="hidden"
                                    accept={ALLOWED_EXTENSIONS.join(',')}
                                    onChange={(e) => {
                                        if (e.target.files?.length) addFiles(e.target.files);
                                        e.target.value = '';
                                    }}
                                />
                                <Upload className="w-5 h-5 text-dark-500 mx-auto mb-2" />
                                <p className="text-xs text-dark-400">
                                    Drag & drop files or <span className="text-teal-400 underline">browse</span>
                                </p>
                                <p className="text-[10px] text-dark-600 mt-1">
                                    PDF, TXT, DOC, images, ZIP • Max 10 MB each • {MAX_FILES} files max
                                </p>
                            </div>

                            {/* File List */}
                            {files.length > 0 && (
                                <div className="space-y-1.5">
                                    {files.map((file, index) => (
                                        <div
                                            key={`${file.name}-${index}`}
                                            className="flex items-center justify-between bg-dark-900 border border-dark-700 rounded-lg px-3 py-2"
                                        >
                                            <div className="flex items-center gap-2 min-w-0">
                                                <Paperclip className="w-3 h-3 text-teal-400 shrink-0" />
                                                <span className="text-xs text-dark-200 truncate">{file.name}</span>
                                                <span className="text-[10px] text-dark-500 shrink-0">{formatFileSize(file.size)}</span>
                                            </div>
                                            <button
                                                onClick={(e) => { e.stopPropagation(); removeFile(index); }}
                                                className="text-dark-500 hover:text-red-400 transition-colors p-0.5"
                                            >
                                                <X className="w-3.5 h-3.5" />
                                            </button>
                                        </div>
                                    ))}
                                    <p className="text-[10px] text-dark-600 text-right">
                                        Total: {formatFileSize(files.reduce((sum, f) => sum + f.size, 0))} / 25 MB
                                    </p>
                                </div>
                            )}
                        </div>
                    )}

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Mail className="w-3 h-3" /> Recipient Email
                            </label>
                            <Input
                                type="email"
                                placeholder="recipient@email.com"
                                value={email}
                                onChange={(e) => {
                                    setEmail(e.target.value);
                                    if (error) setError(null);
                                    if (success) setSuccess(false);
                                }}
                                className="bg-dark-950 border-dark-700 focus:border-teal-500 text-dark-100 placeholder:text-dark-500"
                                aria-invalid={Boolean(error)}
                            />
                        </div>

                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Clock className="w-3 h-3" /> Trigger After
                            </label>
                            <Select
                                value={duration}
                                onChange={(e) => handleDurationChange(Number(e.target.value))}
                                className="bg-dark-950 border-dark-700 text-dark-100"
                            >
                                {timePresets.map(preset => (
                                    <option key={preset.value} value={preset.value}>
                                        {preset.label}
                                    </option>
                                ))}
                            </Select>
                        </div>
                    </div>

                    <div className="space-y-2">
                        <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                            <Clock className="w-3 h-3 text-teal-400" /> Reminders Before Trigger
                        </label>
                        <div className="flex flex-col gap-2 bg-dark-900 border border-dark-700 rounded-lg p-3">
                            {reminders.length > 0 ? (
                                <div className="flex flex-wrap gap-2">
                                    {reminders.map(r => {
                                        const preset = reminderPresets.find(p => p.value === r);
                                        const label = preset ? preset.label : formatMinutes(r);
                                        return (
                                            <div key={r} className="flex items-center gap-1 bg-dark-800 text-dark-200 text-xs px-2 py-1 rounded">
                                                <span>{label}</span>
                                                <button onClick={() => removeReminder(r)} className="text-dark-400 hover:text-red-400">
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
                                            addReminder(Number(e.target.value));
                                            e.target.value = '';
                                        }
                                    }}
                                    className="bg-dark-950 border-dark-700 text-dark-100 text-xs h-8"
                                    value={""}
                                >
                                    <option value="" disabled>Add a reminder...</option>
                                    {reminderPresets.filter(p => !reminders.includes(p.value) && p.value < duration).map(preset => (
                                        <option key={preset.value} value={preset.value}>
                                            {preset.label}
                                        </option>
                                    ))}
                                </Select>
                            </div>
                        </div>
                    </div>

                    {error && (
                        <Alert variant="destructive" className="border-red-500/20 bg-red-500/10">
                            <AlertCircle className="h-4 w-4" />
                            <AlertDescription className="space-y-3">
                                <p>{error.replace(/^SMTP_.*?: /, '')}</p>
                                {smtpError && (
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        className="w-full border-red-500/50 hover:bg-red-500/20 text-red-500 transition-colors mt-2"
                                        onClick={() => setRoute?.('settings')}
                                    >
                                        <SettingsIcon className="w-3.5 h-3.5 mr-2" />
                                        Go to Settings
                                    </Button>
                                )}
                            </AlertDescription>
                        </Alert>
                    )}

                    {success && (
                        <Alert className="border-teal-500/20 bg-teal-500/10">
                            <CheckCircle className="h-4 w-4 text-teal-400" />
                            <AlertDescription className="text-teal-400">
                                Switch activated! Remember to check in regularly.
                            </AlertDescription>
                        </Alert>
                    )}

                    {uploadProgress && (
                        <div className="flex items-center gap-2 text-xs text-teal-400">
                            <Loader2 className="w-3 h-3 animate-spin" />
                            {uploadProgress}
                        </div>
                    )}
                </CardContent>
                <CardFooter>
                    <Button
                        className="w-full bg-teal-600 hover:bg-teal-500 text-white font-medium py-5"
                        onClick={handleCreate}
                        disabled={loading || !message.trim() || !email.trim()}
                    >
                        {loading ? (
                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                        ) : (
                            <Send className="w-4 h-4 mr-2" />
                        )}
                        Activate Switch
                    </Button>
                </CardFooter>
            </Card>

            <div className="text-center text-xs text-dark-500 space-y-1">
                <p>Make sure to send heartbeats from the Dashboard to prevent delivery</p>
            </div>
        </div>
    );
}
