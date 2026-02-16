import { useState, useEffect } from 'react'
import Header from './components/Header'
import CreateSwitch from './components/CreateSwitch'
import Dashboard from './components/Dashboard'
import Settings from './components/Settings'
import VaultLock from './components/VaultLock'
import SecurityBanner from './components/SecurityBanner'
import { apiRequest } from './lib/api'

function App() {
  const [route, setRoute] = useState('home')
  const [authorized, setAuthorized] = useState(false)

  useEffect(() => {
    const path = window.location.pathname;
    // No longer need to handle /view/ routes since we send content in email
  }, []);

  useEffect(() => {
    const checkSession = async () => {
      try {
        const data = await apiRequest('/auth/session');
        setAuthorized(Boolean(data?.authorized));
      } catch {
        setAuthorized(false);
      }
    };
    checkSession();
  }, []);

  const handleUnlock = () => {
    setAuthorized(true);
  };

  const handleLogout = async () => {
    try {
      await apiRequest('/auth/logout', { method: 'POST' });
    } catch {
      // ignore logout errors
    }
    setAuthorized(false);
  };

  const isLocked = !authorized;

  return (
    <div className="min-h-screen">
      <SecurityBanner />
      <Header
        currentRoute={route}
        setRoute={setRoute}
        onLogout={handleLogout}
      />

      <main className="container mx-auto px-3 sm:px-4 pt-20 sm:pt-28 pb-12 sm:pb-16 flex flex-col items-center">
        {isLocked ? (
          <VaultLock onUnlock={handleUnlock} />
        ) : (
          <>
            {route === 'home' && <CreateSwitch setRoute={setRoute} />}
            {route === 'dashboard' && <Dashboard />}
            {route === 'settings' && <Settings />}
          </>
        )}

        <div className="mt-8 sm:mt-12 text-dark-500 text-[10px] sm:text-xs flex items-center gap-3 sm:gap-4">
          <span>&copy; 2026 Aeterna</span>
          <span className="w-1 h-1 rounded-full bg-dark-700" />
          <span>Dead Man's Switch</span>
        </div>
      </main>
    </div>
  )
}

export default App
