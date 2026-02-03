import { Shield, LayoutDashboard, PlusCircle, LogOut, Settings } from 'lucide-react';
import { Button } from "@/components/ui/button"

export default function Header({ currentRoute, setRoute, onLogout }) {
    return (
        <header className="fixed top-0 left-0 right-0 z-50 border-b border-dark-700 bg-dark-950/95 backdrop-blur-sm">
            <div className="container mx-auto px-4 h-14 flex items-center justify-between">
                <div
                    className="flex items-center gap-2.5 cursor-pointer"
                    onClick={() => setRoute('home')}
                >
                    <Shield className="w-5 h-5 text-teal-400" />
                    <span className="font-semibold text-dark-100 tracking-tight">
                        Aeterna
                    </span>
                </div>

                <nav className="flex items-center gap-1">
                    <Button
                        variant="ghost"
                        size="sm"
                        className={`text-dark-400 hover:text-dark-100 hover:bg-dark-800 ${currentRoute === 'home' ? 'bg-dark-800 text-dark-100' : ''}`}
                        onClick={() => setRoute('home')}
                    >
                        <PlusCircle className="w-4 h-4 mr-2" />
                        Create
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className={`text-dark-400 hover:text-dark-100 hover:bg-dark-800 ${currentRoute === 'dashboard' ? 'bg-dark-800 text-dark-100' : ''}`}
                        onClick={() => setRoute('dashboard')}
                    >
                        <LayoutDashboard className="w-4 h-4 mr-2" />
                        Dashboard
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className={`text-dark-400 hover:text-dark-100 hover:bg-dark-800 ${currentRoute === 'settings' ? 'bg-dark-800 text-dark-100' : ''}`}
                        onClick={() => setRoute('settings')}
                    >
                        <Settings className="w-4 h-4 mr-2" />
                        Settings
                    </Button>
                    {onLogout && (
                        <>
                            <div className="w-px h-4 bg-dark-700 mx-2" />
                            <Button
                                variant="ghost"
                                size="icon"
                                className="text-dark-500 hover:text-red-400 hover:bg-dark-800"
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
