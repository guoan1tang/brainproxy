import './Header.css';

interface HeaderProps {
  connected: boolean;
  requestCount: number;
}

export function Header({ connected, requestCount }: HeaderProps) {
  return (
    <header className="header">
      <div className="header-left">
        <span className="header-logo">🧠</span>
        <h1 className="header-title">BrainProxy</h1>
      </div>
      <div className="header-right">
        <span className="header-stat">{requestCount} requests</span>
        <span className={`header-status ${connected ? 'connected' : 'disconnected'}`}>
          {connected ? '● Connected' : '○ Disconnected'}
        </span>
      </div>
    </header>
  );
}
