import { Shield, LayoutDashboard, PlusCircle, LogOut, Settings } from 'lucide-react';
import { Button } from "@/components/ui/button"

export default function Header({ currentRoute, setRoute, onLogout }) {
    return (
        <header className="fixed top-0 left-0 right-0 z-50 border-b border-white/5 bg-slate-950/50 backdrop-blur-xl">
            <div className="container mx-auto px-4 h-16 flex items-center justify-between">
                <div
                    className="flex items-center gap-2 cursor-pointer group"
                    onClick={() => setRoute('home')}
                >
                    <div className="p-2 bg-cyan-500/10 rounded-lg group-hover:bg-cyan-500/20 transition-colors">
                        <Shield className="w-5 h-5 text-cyan-400" />
                    </div>
                    <span className="font-bold text-lg tracking-tight bg-gradient-to-r from-white to-slate-400 bg-clip-text text-transparent">
                        AETERNA
                    </span>
                </div>

                <nav className="flex items-center gap-1">
                    <Button
                        variant="ghost"
                        size="sm"
                        className={`text-slate-400 hover:text-white ${currentRoute === 'home' ? 'bg-white/5 text-white' : ''}`}
                        onClick={() => setRoute('home')}
                    >
                        <PlusCircle className="w-4 h-4 mr-2" />
                        Create
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className={`text-slate-400 hover:text-white ${currentRoute === 'dashboard' ? 'bg-white/5 text-white' : ''}`}
                        onClick={() => setRoute('dashboard')}
                    >
                        <LayoutDashboard className="w-4 h-4 mr-2" />
                        Dashboard
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className={`text-slate-400 hover:text-white ${currentRoute === 'settings' ? 'bg-white/5 text-white' : ''}`}
                        onClick={() => setRoute('settings')}
                    >
                        <Settings className="w-4 h-4 mr-2" />
                        Settings
                    </Button>
                    {onLogout && (
                        <>
                            <div className="w-px h-4 bg-white/10 mx-2" />
                            <Button
                                variant="ghost"
                                size="icon"
                                className="text-slate-600 hover:text-red-400"
                                onClick={onLogout}
                            >
                                <LogOut className="w-4 h-4" />
                            </Button>
                        </>
                    )}
                </nav>
            </div>
        </header>
    );
}
