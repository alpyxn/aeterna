import { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Mail, Server, Save, Loader2, CheckCircle, Eye, EyeOff, TestTube, ChevronDown, ChevronUp, ExternalLink } from 'lucide-react';
import { apiRequest } from "@/lib/api";

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

export default function Settings({ masterKey }) {
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
    const [showGuide, setShowGuide] = useState(false);

    useEffect(() => {
        fetchConfig();
    }, []);

    const fetchConfig = async () => {
        setConfigLoading(true);
        try {
            const data = await apiRequest('/settings', {
                headers: { 'X-Master-Key': masterKey }
            });
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
                headers: {
                    'X-Master-Key': masterKey
                },
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
                headers: {
                    'X-Master-Key': masterKey
                },
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
                <h1 className="text-3xl font-black text-white">Settings</h1>
                <p className="text-slate-500 text-sm">Configure email delivery and system options</p>
            </div>

            {/* SMTP Guide */}
            <Card className="border-slate-800 bg-slate-900/50">
                <button
                    className="w-full p-4 flex items-center justify-between text-left"
                    onClick={() => setShowGuide(!showGuide)}
                >
                    <div>
                        <h3 className="text-sm font-semibold text-white">Quick Setup Guide</h3>
                        <p className="text-xs text-slate-500">Pre-configured settings for popular email providers</p>
                    </div>
                    {showGuide ? <ChevronUp className="w-5 h-5 text-slate-400" /> : <ChevronDown className="w-5 h-5 text-slate-400" />}
                </button>
                {showGuide && (
                    <div className="px-4 pb-4 space-y-2">
                        {SMTP_GUIDES.map(guide => (
                            <div key={guide.name} className="flex items-center justify-between p-3 bg-slate-950 rounded-lg border border-slate-800">
                                <div className="flex-1">
                                    <div className="flex items-center gap-2">
                                        <span className="font-medium text-sm text-white">{guide.name}</span>
                                        <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${guide.security === 'SSL' ? 'bg-purple-500/20 text-purple-400' : 'bg-cyan-500/20 text-cyan-400'}`}>
                                            {guide.security}
                                        </span>
                                        <a href={guide.link} target="_blank" rel="noopener noreferrer" className="text-cyan-400 hover:text-cyan-300">
                                            <ExternalLink className="w-3 h-3" />
                                        </a>
                                    </div>
                                    <p className="text-xs text-slate-500 mt-0.5">{guide.host}:{guide.port}</p>
                                    <p className="text-xs text-slate-600 mt-1">{guide.note}</p>
                                </div>
                                <Button
                                    size="sm"
                                    variant="outline"
                                    className="border-slate-700 hover:bg-slate-800 text-xs"
                                    onClick={() => applyGuide(guide)}
                                >
                                    Apply
                                </Button>
                            </div>
                        ))}
                    </div>
                )}
            </Card>

            <Card className="glowing-card border-slate-800">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-lg">
                        <Mail className="w-5 h-5 text-cyan-400" />
                        SMTP Configuration
                    </CardTitle>
                    <CardDescription>
                        Configure your email server for sending triggered messages
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-slate-500 uppercase tracking-wider flex items-center gap-2">
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
                                className="bg-slate-950 border-slate-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-slate-500 uppercase tracking-wider">
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
                                className="bg-slate-950 border-slate-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-slate-500 uppercase tracking-wider">
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
                                className="bg-slate-950 border-slate-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-slate-500 uppercase tracking-wider">
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
                                    className="bg-slate-950 border-slate-800 pr-10"
                                    aria-invalid={Boolean(error)}
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowPassword(!showPassword)}
                                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
                                >
                                    {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                </button>
                            </div>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-slate-500 uppercase tracking-wider">
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
                                className="bg-slate-950 border-slate-800"
                                aria-invalid={Boolean(error)}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-xs font-bold text-slate-500 uppercase tracking-wider">
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
                                className="bg-slate-950 border-slate-800"
                                aria-invalid={Boolean(error)}
                            />
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
                <CardFooter className="flex gap-2">
                    <Button
                        variant="outline"
                        className="border-slate-700 hover:bg-slate-800"
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
                        className="flex-1 bg-cyan-600 hover:bg-cyan-500"
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
