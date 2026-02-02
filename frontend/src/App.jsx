import { useState, useEffect } from 'react'
import Header from './components/Header'
import CreateSwitch from './components/CreateSwitch'
import Dashboard from './components/Dashboard'
import Settings from './components/Settings'
import VaultLock from './components/VaultLock'

function App() {
  const [route, setRoute] = useState('home')
  const [masterKey, setMasterKey] = useState(sessionStorage.getItem('aeterna_master_key'))

  useEffect(() => {
    const path = window.location.pathname;
    // No longer need to handle /view/ routes since we send content in email
  }, []);

  const handleUnlock = (key) => {
    setMasterKey(key);
    sessionStorage.setItem('aeterna_master_key', key);
  };

  const handleLogout = () => {
    setMasterKey(null);
    sessionStorage.removeItem('aeterna_master_key');
  };

  const isLocked = !masterKey;

  return (
    <div className="min-h-screen">
      <Header
        currentRoute={route}
        setRoute={setRoute}
        onLogout={handleLogout}
      />

      <main className="container mx-auto px-4 pt-32 pb-16 flex flex-col items-center">
        {isLocked ? (
          <VaultLock onUnlock={handleUnlock} />
        ) : (
          <>
            {route === 'home' && <CreateSwitch masterKey={masterKey} />}
            {route === 'dashboard' && <Dashboard masterKey={masterKey} />}
            {route === 'settings' && <Settings masterKey={masterKey} />}
          </>
        )}

        <div className="mt-12 text-slate-600 text-xs flex items-center gap-4">
          <span>&copy; 2026 Aeterna Project</span>
          <span className="w-1 h-1 rounded-full bg-slate-800" />
          <span>Dead Man's Switch</span>
          <span className="w-1 h-1 rounded-full bg-slate-800" />
          <span>Control Center {masterKey ? 'Authorized' : 'Restricted'}</span>
        </div>
      </main>
    </div>
  )
}

export default App
