import { useState } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Lock, Mail, Clock, Loader2, AlertCircle, CheckCircle, Send } from 'lucide-react';
import { Select } from "@/components/ui/select"
import { apiRequest } from "@/lib/api"

export default function CreateSwitch() {
    const [message, setMessage] = useState('');
    const [email, setEmail] = useState('');
    const [duration, setDuration] = useState(1440);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);
    const [success, setSuccess] = useState(false);

    const timePresets = [
        { label: '1 Minute (Debug)', value: 1 },
        { label: '15 Minutes (Test)', value: 15 },
        { label: '1 Hour', value: 60 },
        { label: '1 Day', value: 1440 },
        { label: '3 Days', value: 4320 },
        { label: '1 Week', value: 10080 },
        { label: '2 Weeks', value: 20160 },
        { label: '1 Month', value: 43200 },
    ];

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

        try {
            await apiRequest('/messages', {
                method: 'POST',
                body: JSON.stringify({
                    content: message,
                    recipient_email: email,
                    trigger_duration: duration
                })
            });

            setSuccess(true);
            setMessage('');
            setEmail('');

            setTimeout(() => setSuccess(false), 5000);
        } catch (e) {
            setError(e.message);
        } finally {
            setLoading(false);
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
                                onChange={(e) => setDuration(Number(e.target.value))}
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

                    {error && (
                        <Alert variant="destructive" className="border-red-500/20 bg-red-500/10">
                            <AlertCircle className="h-4 w-4" />
                            <AlertDescription>{error}</AlertDescription>
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
                <p>Your message will be stored securely on our servers</p>
                <p>Make sure to send heartbeats from the Dashboard to prevent delivery</p>
            </div>
        </div>
    );
}
