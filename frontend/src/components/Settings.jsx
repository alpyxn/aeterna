import { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Mail, Server, Save, Loader2, CheckCircle, Eye, EyeOff, TestTube, ChevronDown, ChevronUp, ExternalLink, Trash2 } from 'lucide-react';
import { apiRequest } from "@/lib/api";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog"

const SMTP_GUIDES = [
    {
        name: 'Gmail',
        host: 'smtp.gmail.com',
        port: '587',
        security: 'STARTTLS',
        note: 'Requires App Password. Go to Google Account → Security → 2-Step Verification → App passwords',
        link: 'https://support.google.com/accounts/answer/185833'
    },
    {
        name: 'Yandex',
        host: 'smtp.yandex.com',
        port: '465',
        security: 'SSL',
        note: 'Use your Yandex email and password. Enable IMAP/SMTP in settings.',
        link: 'https://yandex.com/support/mail/mail-clients/others.html'
    },
    {
        name: 'Brevo (Sendinblue)',
        host: 'smtp-relay.brevo.com',
        port: '587',
        security: 'STARTTLS',
        note: 'Use your Brevo login email and SMTP key (not password)',
        link: 'https://app.brevo.com/settings/keys/smtp'
    },
    {
        name: 'Mailgun',
        host: 'smtp.mailgun.org',
        port: '587',
        security: 'STARTTLS',
        note: 'Use your Mailgun domain credentials',
        link: 'https://app.mailgun.com/app/sending/domains'
    },
    {
        name: 'SendGrid',
        host: 'smtp.sendgrid.net',
        port: '587',
        security: 'STARTTLS',
        note: 'Username: apikey, Password: your API key',
        link: 'https://app.sendgrid.com/settings/api_keys'
    },
    {
        name: 'Outlook/Office365',
        host: 'smtp.office365.com',
        port: '587',
        security: 'STARTTLS',
        note: 'Use your Microsoft account email and password',
        link: 'https://support.microsoft.com/en-us/office/pop-imap-and-smtp-settings'
    },
    {
        name: 'Zoho',
        host: 'smtp.zoho.com',
        port: '465',
        security: 'SSL',
        note: 'Use Zoho email and password. Enable SMTP in settings.',
        link: 'https://www.zoho.com/mail/help/zoho-smtp.html'
    }
];

export default function Settings() {
    const [config, setConfig] = useState({
        smtp_host: '',
        smtp_port: '587',
        smtp_user: '',
        smtp_pass: '',
        smtp_from: '',
        smtp_from_name: 'Aeterna Vault'
    });
    const [configLoading, setConfigLoading] = useState(true);
    const [loading, setLoading] = useState(false);
    const [testLoading, setTestLoading] = useState(false);
    const [saved, setSaved] = useState(false);
    const [testSuccess, setTestSuccess] = useState(false);
    const [error, setError] = useState(null);
    const [showPassword, setShowPassword] = useState(false);
    const [webhooks, setWebhooks] = useState([]);
    const [webhookLoading, setWebhookLoading] = useState(false);
    const [showWebhookSecret, setShowWebhookSecret] = useState(false);
    const [showGuide, setShowGuide] = useState(false);

    useEffect(() => {
        fetchConfig();
        fetchWebhooks();
    }, []);

    const fetchConfig = async () => {
        setConfigLoading(true);
        try {
            const data = await apiRequest('/settings');
            if (data) {
                setConfig(prev => ({ ...prev, ...data }));
            }
        } catch (e) {
            console.error('Failed to fetch config');
            setError('Failed to load settings');
        } finally {
            setConfigLoading(false);
        }
    };

    const fetchWebhooks = async () => {
        setWebhookLoading(true);
        try {
            const data = await apiRequest('/webhooks');
            setWebhooks(Array.isArray(data) ? data : []);
        } catch (e) {
            setError(e.message || 'Failed to load webhooks');
        } finally {
            setWebhookLoading(false);
        }
    };

    const addWebhook = () => {
        setWebhooks(prev => ([
            ...prev,
            { id: null, url: '', secret: '', enabled: true, isNew: true }
        ]));
    };

    const updateWebhook = (index, patch, isDirty = true) => {
        setWebhooks(prev => prev.map((item, i) => (i === index ? { ...item, ...patch, isDirty } : item)));
    };

    const saveWebhook = async (item, index) => {
        try {
            if (!item.url) {
                setError('Webhook URL is required');
                return;
            }
            if (item.id) {
                const updated = await apiRequest(`/webhooks/${item.id}`, {
                    method: 'PUT',
                    body: JSON.stringify({ url: item.url, secret: item.secret, enabled: item.enabled })
                });
                updateWebhook(index, { ...updated, isNew: false, confirmingDelete: false }, false);
            } else {
                const created = await apiRequest('/webhooks', {
                    method: 'POST',
                    body: JSON.stringify({ url: item.url, secret: item.secret, enabled: item.enabled })
                });
                updateWebhook(index, { ...created, isNew: false, confirmingDelete: false }, false);
            }
            setError(null);
        } catch (e) {
            setError(e.message || 'Failed to save webhook');
        }
    };

    const deleteWebhook = async (index) => {
        const item = webhooks[index];
        try {
            if (item.id) {
                await apiRequest(`/webhooks/${item.id}`, {
                    method: 'DELETE'
                });
            }
            setWebhooks(prev => prev.filter((_, i) => i !== index));
        } catch (e) {
            setError(e.message || 'Failed to delete webhook');
        }
    };

    const applyGuide = (guide) => {
        setConfig(prev => ({
            ...prev,
            smtp_host: guide.host,
            smtp_port: guide.port
        }));
    };

    const handleSave = async () => {
        setLoading(true);
        setError(null);
        setSaved(false);
        try {
            await apiRequest('/settings', {
                method: 'POST',
                body: JSON.stringify(config)
            });
            setSaved(true);
            setTimeout(() => setSaved(false), 3000);
        } catch (e) {
            setError(e.message);
        } finally {
            setLoading(false);
        }
    };

    const handleTest = async () => {
        if (!config.smtp_host || !config.smtp_port || !config.smtp_user || !config.smtp_pass) {
            setError('SMTP host, port, username, and password are required to test connection');
            return;
        }
        setTestLoading(true);
        setError(null);
        setTestSuccess(false);
        try {
            await apiRequest('/settings/test', {
                method: 'POST',
                body: JSON.stringify(config)
            });
            setTestSuccess(true);
            setTimeout(() => setTestSuccess(false), 3000);
        } catch (e) {
            setError(e.message);
        } finally {
            setTestLoading(false);
        }
    };

    return (
        <div className="w-full max-w-2xl space-y-6">
            <div>
                <h1 className="text-2xl font-semibold text-dark-100">Settings</h1>
                <p className="text-dark-400 text-sm">Configure email delivery and system options</p>
            </div>

            {/* SMTP Guide */}
            <Card className="border-dark-700 bg-dark-900">
                <button
                    className="w-full p-4 flex items-center justify-between text-left"
                    onClick={() => setShowGuide(!showGuide)}
                >
                    <div>
                        <h3 className="text-sm font-medium text-dark-100">Quick Setup Guide</h3>
                        <p className="text-xs text-dark-500">Pre-configured settings for popular email providers</p>
                    </div>
                    {showGuide ? <ChevronUp className="w-5 h-5 text-dark-400" /> : <ChevronDown className="w-5 h-5 text-dark-400" />}
                </button>
                {showGuide && (
                    <div className="px-4 pb-4 space-y-2">
                        {SMTP_GUIDES.map(guide => (
                            <div key={guide.name} className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 bg-dark-950 rounded-lg border border-dark-800">
                                <div className="flex-1">
                                    <div className="flex items-center gap-2">
                                        <span className="font-medium text-sm text-dark-100">{guide.name}</span>
                                        <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${guide.security === 'SSL' ? 'bg-purple-500/20 text-purple-400' : 'bg-teal-500/20 text-teal-400'}`}>
                                            {guide.security}
                                        </span>
                                        <a href={guide.link} target="_blank" rel="noopener noreferrer" className="text-teal-400 hover:text-teal-300">
                                            <ExternalLink className="w-3 h-3" />
                                        </a>
                                    </div>
                                    <p className="text-xs text-dark-400 mt-0.5">{guide.host}:{guide.port}</p>
                                    <p className="text-xs text-dark-500 mt-1">{guide.note}</p>
                                </div>
                                <Button
                                    size="sm"
                                    variant="outline"
                                    className="border-dark-700 hover:bg-dark-800 text-xs"
                                    onClick={() => applyGuide(guide)}
                                >
                                    Apply
                                </Button>
                            </div>
                        ))}
                    </div>
                )}
            </Card>

            <Card className="glowing-card">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-base font-medium">
                        <Mail className="w-4 h-4 text-teal-400" />
                        SMTP Configuration
                    </CardTitle>
                    <CardDescription className="text-dark-400">
                        Configure your email server for sending triggered messages
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <label className="text-xs font-medium text-dark-400 flex items-center gap-2">
                                <Server className="w-3 h-3" /> SMTP Host
                            </label>
                            <Input
                                placeholder="smtp.gmail.com"
                                value={config.smtp_host}
                                onChange={(e) => {
                                    setConfig({ ...config, smtp_host: e.target.value });
                                    if (error) setError(null);
                                    if (saved) setSaved(false);
                                    if (testSuccess) setTestSuccess(false);
                                }}
                                className="bg-dark-950 border-dark-700 text-dark-100 placeholder:text-dark-500"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-dark-500 uppercase tracking-wider">
                                SMTP Port
                            </label>
                            <Input
                                placeholder="587"
                                value={config.smtp_port}
                                onChange={(e) => {
                                    setConfig({ ...config, smtp_port: e.target.value });
                                    if (error) setError(null);
                                    if (saved) setSaved(false);
                                    if (testSuccess) setTestSuccess(false);
                                }}
                                className="bg-dark-950 border-dark-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-dark-500 uppercase tracking-wider">
                                Username / Email
                            </label>
                            <Input
                                placeholder="your@email.com"
                                value={config.smtp_user}
                                onChange={(e) => {
                                    setConfig({ ...config, smtp_user: e.target.value });
                                    if (error) setError(null);
                                    if (saved) setSaved(false);
                                    if (testSuccess) setTestSuccess(false);
                                }}
                                className="bg-dark-950 border-dark-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-dark-500 uppercase tracking-wider">
                                Password / App Password
                            </label>
                            <div className="relative">
                                <Input
                                    type={showPassword ? "text" : "password"}
                                    placeholder="••••••••"
                                    value={config.smtp_pass}
                                    onChange={(e) => {
                                        setConfig({ ...config, smtp_pass: e.target.value });
                                        if (error) setError(null);
                                        if (saved) setSaved(false);
                                        if (testSuccess) setTestSuccess(false);
                                    }}
                                    className="bg-dark-950 border-dark-800 pr-10"
                                    aria-invalid={Boolean(error)}
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowPassword(!showPassword)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-500 hover:text-dark-300"
                                >
                                    {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                </button>
                            </div>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-dark-500 uppercase tracking-wider">
                                From Email
                            </label>
                            <Input
                                placeholder="noreply@yourdomain.com"
                                value={config.smtp_from}
                                onChange={(e) => {
                                    setConfig({ ...config, smtp_from: e.target.value });
                                    if (error) setError(null);
                                    if (saved) setSaved(false);
                                    if (testSuccess) setTestSuccess(false);
                                }}
                                className="bg-dark-950 border-dark-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-dark-500 uppercase tracking-wider">
                                From Name
                            </label>
                            <Input
                                placeholder="Aeterna Vault"
                                value={config.smtp_from_name}
                                onChange={(e) => {
                                    setConfig({ ...config, smtp_from_name: e.target.value });
                                    if (error) setError(null);
                                    if (saved) setSaved(false);
                                    if (testSuccess) setTestSuccess(false);
                                }}
                                className="bg-dark-950 border-dark-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                    </div>

                    <div className="pt-2 border-t border-dark-800/70" />

                    <div className="space-y-3">
                        <div className="flex items-center justify-between">
                            <div>
                                <div className="text-sm font-semibold text-white">Webhooks</div>
                                <div className="text-xs text-dark-500 mt-1">
                                    When any of your switches are triggered, all enabled webhooks will be executed.
                                </div>
                            </div>
                            <Button
                                variant="outline"
                                size="sm"
                                className="border-dark-700 hover:bg-dark-800"
                                onClick={addWebhook}
                                disabled={webhookLoading}
                            >
                                Add Webhook
                            </Button>
                        </div>

                        {webhookLoading && (
                            <div className="text-xs text-dark-500">Loading webhooks...</div>
                        )}

                        {webhooks.length === 0 && !webhookLoading && (
                            <div className="text-xs text-dark-500">No webhooks configured.</div>
                        )}

                        <div className="space-y-3">
                            {webhooks.map((item, index) => (
                                <div key={item.id ?? `new-${index}`} className="rounded-lg border border-dark-800 bg-dark-950/60 p-3 space-y-3">
                                    <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2">
                                        <div className="flex items-center gap-3">
                                            <label className="flex items-center gap-2 text-xs text-dark-400">
                                                <input
                                                    type="checkbox"
                                                    checked={Boolean(item.enabled)}
                                                    onChange={(e) => updateWebhook(index, { enabled: e.target.checked })}
                                                    className="h-4 w-4 accent-teal-400"
                                                />
                                                Enabled
                                            </label>
                                            {item.isDirty && (
                                                <span className="text-[10px] bg-amber-500/10 text-amber-400 px-2 py-0.5 rounded border border-amber-500/20 animate-pulse">
                                                    Unsaved Changes
                                                </span>
                                            )}
                                        </div>
                                        <div className="flex flex-wrap items-center gap-2">
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                className={`transition-all ${item.isDirty ? 'border-teal-500 bg-teal-500/10 text-teal-400 hover:bg-teal-500/20' : 'border-dark-700 hover:bg-dark-800'}`}
                                                onClick={() => saveWebhook(item, index)}
                                            >
                                                {item.isDirty ? 'Save Changes' : 'Save'}
                                            </Button>
                                            <AlertDialog>
                                                <AlertDialogTrigger asChild>
                                                    <Button
                                                        variant="outline"
                                                        size="sm"
                                                        className="border-red-500/40 hover:bg-red-500/10 text-red-400"
                                                    >
                                                        <Trash2 className="w-3.5 h-3.5 mr-1" />
                                                        Delete
                                                    </Button>
                                                </AlertDialogTrigger>
                                                <AlertDialogContent>
                                                    <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
                                                    <AlertDialogDescription>
                                                        This will permanently delete the webhook for <strong>{item.url || 'this endpoint'}</strong>.
                                                        This action cannot be undone.
                                                    </AlertDialogDescription>
                                                    <AlertDialogFooter>
                                                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                                                        <AlertDialogAction
                                                            onClick={() => deleteWebhook(index)}
                                                            className="bg-red-600 hover:bg-red-700"
                                                        >
                                                            Delete Webhook
                                                        </AlertDialogAction>
                                                    </AlertDialogFooter>
                                                </AlertDialogContent>
                                            </AlertDialog>
                                        </div>
                                    </div>

                                    <div className="space-y-2">
                                        <label className="text-xs font-bold text-dark-500 uppercase tracking-wider">
                                            Webhook URL
                                        </label>
                                        <Input
                                            placeholder="https://example.com/webhook"
                                            value={item.url}
                                            onChange={(e) => updateWebhook(index, { url: e.target.value })}
                                            className="bg-dark-950 border-dark-800"
                                            aria-invalid={Boolean(error)}
                                        />
                                    </div>

                                    <div className="space-y-2">
                                        <label className="text-xs font-bold text-dark-500 uppercase tracking-wider">
                                            Webhook Secret (optional)
                                        </label>
                                        <div className="relative">
                                            <Input
                                                type={showWebhookSecret ? "text" : "password"}
                                                placeholder="••••••••"
                                                value={item.secret}
                                                onChange={(e) => updateWebhook(index, { secret: e.target.value })}
                                                className="bg-dark-950 border-dark-800 pr-10"
                                                aria-invalid={Boolean(error)}
                                            />
                                            <button
                                                type="button"
                                                onClick={() => setShowWebhookSecret(!showWebhookSecret)}
                                                className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-500 hover:text-dark-300"
                                            >
                                                {showWebhookSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    {error && (
                        <Alert variant="destructive">
                            <AlertDescription>{error}</AlertDescription>
                        </Alert>
                    )}

                    {configLoading && (
                        <Alert>
                            <AlertDescription>Loading settings...</AlertDescription>
                        </Alert>
                    )}

                    {saved && (
                        <Alert className="border-green-500/30 bg-green-500/10">
                            <CheckCircle className="h-4 w-4 text-green-400" />
                            <AlertDescription className="text-green-400">
                                Settings saved successfully!
                            </AlertDescription>
                        </Alert>
                    )}

                    {testSuccess && (
                        <Alert className="border-green-500/30 bg-green-500/10">
                            <CheckCircle className="h-4 w-4 text-green-400" />
                            <AlertDescription className="text-green-400">
                                SMTP connection successful!
                            </AlertDescription>
                        </Alert>
                    )}
                </CardContent>
                <CardFooter className="flex flex-col sm:flex-row gap-2">
                    <Button
                        variant="outline"
                        className="border-dark-700 hover:bg-dark-800"
                        onClick={handleTest}
                        disabled={
                            testLoading ||
                            configLoading ||
                            !config.smtp_host ||
                            !config.smtp_port ||
                            !config.smtp_user ||
                            !config.smtp_pass
                        }
                    >
                        {testLoading ? (
                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                        ) : (
                            <TestTube className="w-4 h-4 mr-2" />
                        )}
                        Test Connection
                    </Button>
                    <Button
                        className="flex-1 bg-teal-600 hover:bg-teal-500"
                        onClick={handleSave}
                        disabled={loading || configLoading}
                    >
                        {loading ? (
                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                        ) : (
                            <Save className="w-4 h-4 mr-2" />
                        )}
                        Save Settings
                    </Button>
                </CardFooter>
            </Card>
        </div>
    );
}
