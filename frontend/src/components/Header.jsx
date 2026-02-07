import { useState } from 'react';
import { Shield, LayoutDashboard, PlusCircle, LogOut, Settings, Menu, X } from 'lucide-react';
import { Button } from "@/components/ui/button"

export default function Header({ currentRoute, setRoute, onLogout }) {
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

    const NavButton = ({ route, icon: Icon, label }) => (
        <Button
            variant="ghost"
            size="sm"
            className={`text-dark-400 hover:text-dark-100 hover:bg-dark-800 w-full md:w-auto justify-start md:justify-center ${currentRoute === route ? 'bg-dark-800 text-dark-100' : ''}`}
            onClick={() => {
                setRoute(route);
                setMobileMenuOpen(false);
            }}
        >
            <Icon className="w-4 h-4 mr-2" />
            {label}
        </Button>
    );

    return (
        <header className="fixed top-0 left-0 right-0 z-50 border-b border-dark-700 bg-dark-950/95 backdrop-blur-sm">
            <div className="container mx-auto px-4 h-14 flex items-center justify-between">
                <div
                    className="flex items-center gap-2.5 cursor-pointer"
                    onClick={() => {
                        setRoute('home');
                        setMobileMenuOpen(false);
                    }}
                >
                    <Shield className="w-5 h-5 text-teal-400" />
                    <span className="font-semibold text-dark-100 tracking-tight">
                        Aeterna
                    </span>
                </div>

                {/* Desktop Navigation */}
                <nav className="hidden md:flex items-center gap-1">
                    <NavButton route="home" icon={PlusCircle} label="Create" />
                    <NavButton route="dashboard" icon={LayoutDashboard} label="Dashboard" />
                    <NavButton route="settings" icon={Settings} label="Settings" />
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

                {/* Mobile Menu Button */}
                <Button
                    variant="ghost"
                    size="icon"
                    className="md:hidden text-dark-400 hover:text-dark-100 hover:bg-dark-800"
                    onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                >
                    {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
                </Button>
            </div>

            {/* Mobile Navigation Dropdown */}
            {mobileMenuOpen && (
                <div className="md:hidden border-t border-dark-700 bg-dark-950/98 backdrop-blur-sm">
                    <nav className="container mx-auto px-4 py-3 flex flex-col gap-1">
                        <NavButton route="home" icon={PlusCircle} label="Create" />
                        <NavButton route="dashboard" icon={LayoutDashboard} label="Dashboard" />
                        <NavButton route="settings" icon={Settings} label="Settings" />
                        {onLogout && (
                            <>
                                <div className="h-px bg-dark-700 my-2" />
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    className="text-dark-500 hover:text-red-400 hover:bg-dark-800 w-full justify-start"
                                    onClick={() => {
                                        onLogout();
                                        setMobileMenuOpen(false);
                                    }}
                                >
                                    <LogOut className="w-4 h-4 mr-2" />
                                    Logout
                                </Button>
                            </>
                        )}
                    </nav>
                </div>
            )}
        </header>
    );
}
