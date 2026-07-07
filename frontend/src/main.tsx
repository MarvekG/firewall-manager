import React, { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';

type RuntimeInfo = {
  tlsEnabled: boolean;
  publicUrl: string;
  allowInsecureRemote: boolean;
  version: string;
};

type User = {
  username: string;
};

type PortRule = {
  port: number;
  protocol: 'tcp' | 'udp';
  source?: string;
  description?: string;
};

type FirewallState = {
  osType: string;
  backend: string;
  serviceEnabled: boolean;
  serviceRunning: boolean;
  defaultIncomingPolicy: string;
  openPorts: PortRule[];
  loadedAt: string;
};

type Locale = 'zh-CN' | 'en-US';

const messages: Record<Locale, Record<string, string>> = {
  'zh-CN': {
    title: 'Firewall Manager',
    subtitle: '服务器防火墙端口管理',
    login: '登录',
    username: '用户名',
    password: '密码',
    logout: '退出登录',
    dashboard: '防火墙控制台',
    openPort: '打开端口',
    close: '关闭',
    refresh: '刷新',
    port: '端口',
    protocol: '协议',
    source: '来源',
    description: '描述',
    actions: '操作',
    systemSummary: '系统摘要',
    openPorts: '已开放端口',
    portInput: '端口号',
    tlsWarning: '当前未启用 HTTPS，请仅在可信网络使用。',
    localHttp: '当前为本机 HTTP 模式。',
    invalidLogin: '用户名或密码错误',
    loading: '加载中...',
    noPorts: '当前没有由系统识别的开放端口。',
    confirmClose: '确认关闭端口',
    confirmCloseBody: '关闭该端口可能导致依赖它的服务无法从外部访问。',
    cancel: '取消',
    confirm: '确认关闭',
  },
  'en-US': {
    title: 'Firewall Manager',
    subtitle: 'Server firewall port management',
    login: 'Sign in',
    username: 'Username',
    password: 'Password',
    logout: 'Sign out',
    dashboard: 'Firewall Console',
    openPort: 'Open port',
    close: 'Close',
    refresh: 'Refresh',
    port: 'Port',
    protocol: 'Protocol',
    source: 'Source',
    description: 'Description',
    actions: 'Actions',
    systemSummary: 'System Summary',
    openPorts: 'Open Ports',
    portInput: 'Port',
    tlsWarning: 'HTTPS is disabled. Use only on a trusted network.',
    localHttp: 'Running in local HTTP mode.',
    invalidLogin: 'Invalid username or password',
    loading: 'Loading...',
    noPorts: 'No recognized open ports.',
    confirmClose: 'Close port?',
    confirmCloseBody: 'Closing this port may make dependent services unavailable externally.',
    cancel: 'Cancel',
    confirm: 'Close port',
  },
};

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    ...init,
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body?.error?.code ?? `HTTP_${response.status}`);
  }
  return response.json() as Promise<T>;
}

function App() {
  const [locale, setLocale] = useState<Locale>('zh-CN');
  const [runtime, setRuntime] = useState<RuntimeInfo | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [authChecked, setAuthChecked] = useState(false);

  const t = messages[locale];

  useEffect(() => {
    api<RuntimeInfo>('/api/runtime').then(setRuntime).catch(() => undefined);
    api<User>('/api/auth/me')
      .then((me) => setUser(me))
      .catch(() => setUser(null))
      .finally(() => setAuthChecked(true));
  }, []);

  if (!authChecked) {
    return <FullPageMessage text={t.loading} />;
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <SecurityBanner runtime={runtime} t={t} />
      {user ? (
        <Dashboard user={user} locale={locale} setLocale={setLocale} setUser={setUser} t={t} />
      ) : (
        <Login locale={locale} setLocale={setLocale} setUser={setUser} t={t} />
      )}
    </div>
  );
}

function SecurityBanner({ runtime, t }: { runtime: RuntimeInfo | null; t: Record<string, string> }) {
  if (!runtime || runtime.tlsEnabled) return null;
  const host = window.location.hostname;
  const loopback = host === 'localhost' || host === '127.0.0.1' || host === '::1';
  return (
    <div className={loopback ? 'bg-amber-500/15 px-4 py-2 text-sm text-amber-100' : 'bg-red-500/20 px-4 py-2 text-sm text-red-100'}>
      {loopback ? t.localHttp : t.tlsWarning}
    </div>
  );
}

function Login({ locale, setLocale, setUser, t }: { locale: Locale; setLocale: (locale: Locale) => void; setUser: (user: User) => void; t: Record<string, string> }) {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError('');
    try {
      await api<{ ok: boolean }>('/api/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) });
      const me = await api<User>('/api/auth/me');
      setUser(me);
    } catch {
      setError(t.invalidLogin);
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="mx-auto grid min-h-screen max-w-6xl grid-cols-1 gap-8 px-6 py-12 md:grid-cols-[1.1fr_0.9fr] md:items-center">
      <section className="space-y-6">
        <div className="inline-flex rounded-full border border-cyan-400/30 px-3 py-1 text-sm text-cyan-200">{t.subtitle}</div>
        <h1 className="text-5xl font-bold tracking-tight md:text-6xl">{t.title}</h1>
        <p className="max-w-xl text-lg text-slate-300">管理本机防火墙端口。请确保只在可信网络中访问，并谨慎开放公网端口。</p>
      </section>
      <form onSubmit={submit} className="rounded-3xl border border-white/10 bg-white/10 p-8 shadow-2xl backdrop-blur">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-2xl font-semibold">{t.login}</h2>
          <LocaleSwitch locale={locale} setLocale={setLocale} />
        </div>
        <label className="mb-2 block text-sm text-slate-300" htmlFor="username">{t.username}</label>
        <input id="username" className="mb-4 w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3 outline-none ring-cyan-400 focus:ring-2" value={username} onChange={(e) => setUsername(e.target.value)} disabled={loading} />
        <label className="mb-2 block text-sm text-slate-300" htmlFor="password">{t.password}</label>
        <input id="password" type="password" className="mb-6 w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3 outline-none ring-cyan-400 focus:ring-2" value={password} onChange={(e) => setPassword(e.target.value)} disabled={loading} />
        {error && <div className="mb-4 rounded-xl border border-red-400/40 bg-red-500/10 px-4 py-3 text-sm text-red-100">{error}</div>}
        <button className="w-full rounded-xl bg-cyan-400 px-4 py-3 font-semibold text-slate-950 transition hover:bg-cyan-300 disabled:opacity-60" disabled={loading}>{loading ? t.loading : t.login}</button>
      </form>
    </main>
  );
}

function Dashboard({ user, locale, setLocale, setUser, t }: { user: User; locale: Locale; setLocale: (locale: Locale) => void; setUser: (user: User | null) => void; t: Record<string, string> }) {
  const [state, setState] = useState<FirewallState | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [port, setPort] = useState('');
  const [protocol, setProtocol] = useState<'tcp' | 'udp'>('tcp');
  const [closing, setClosing] = useState<PortRule | null>(null);

  async function loadState() {
    setLoading(true);
    setError('');
    try {
      setState(await api<FirewallState>('/api/firewall/state'));
    } catch (err) {
      setError(String(err instanceof Error ? err.message : err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadState();
  }, []);

  async function logout() {
    await api('/api/auth/logout', { method: 'POST' }).catch(() => undefined);
    setUser(null);
  }

  async function openPort(event: React.FormEvent) {
    event.preventDefault();
    const parsed = Number(port);
    if (!Number.isInteger(parsed) || parsed < 1 || parsed > 65535) {
      setError('PORT_INVALID');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const result = await api<{ state: FirewallState }>('/api/firewall/ports', { method: 'POST', body: JSON.stringify({ port: parsed, protocol }) });
      setState(result.state);
      setPort('');
    } catch (err) {
      setError(String(err instanceof Error ? err.message : err));
    } finally {
      setLoading(false);
    }
  }

  async function closePort(rule: PortRule) {
    setLoading(true);
    setError('');
    try {
      const result = await api<{ state: FirewallState }>(`/api/firewall/ports/${rule.protocol}/${rule.port}`, { method: 'DELETE' });
      setState(result.state);
      setClosing(null);
    } catch (err) {
      setError(String(err instanceof Error ? err.message : err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="mx-auto max-w-7xl px-4 py-6 md:px-8">
      <header className="mb-8 flex flex-col gap-4 rounded-3xl border border-white/10 bg-white/10 p-5 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-3xl font-bold">{t.dashboard}</h1>
          <p className="text-slate-300">{user.username}</p>
        </div>
        <div className="flex flex-wrap gap-3">
          <LocaleSwitch locale={locale} setLocale={setLocale} />
          <button onClick={loadState} className="rounded-xl border border-white/10 px-4 py-2 hover:bg-white/10">{t.refresh}</button>
          <button onClick={logout} className="rounded-xl bg-white px-4 py-2 font-semibold text-slate-950">{t.logout}</button>
        </div>
      </header>

      {error && <div className="mb-6 rounded-2xl border border-red-400/40 bg-red-500/10 p-4 text-red-100">{error}</div>}
      {loading && !state ? <FullPageMessage text={t.loading} /> : null}
      {state ? (
        <div className="grid gap-6 lg:grid-cols-[1.4fr_0.8fr]">
          <section className="space-y-6">
            <Summary state={state} t={t} />
            <PortsTable ports={state.openPorts} onClose={setClosing} t={t} />
          </section>
          <aside className="rounded-3xl border border-white/10 bg-white/10 p-6">
            <h2 className="mb-4 text-xl font-semibold">{t.openPort}</h2>
            <form onSubmit={openPort} className="space-y-4">
              <label className="block text-sm text-slate-300" htmlFor="port">{t.portInput}</label>
              <input id="port" inputMode="numeric" className="w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3 outline-none ring-cyan-400 focus:ring-2" value={port} onChange={(e) => setPort(e.target.value)} />
              <label className="block text-sm text-slate-300" htmlFor="protocol">{t.protocol}</label>
              <select id="protocol" className="w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3" value={protocol} onChange={(e) => setProtocol(e.target.value as 'tcp' | 'udp')}>
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
              </select>
              <div className="rounded-xl bg-slate-950 p-3 text-sm text-slate-300">{port ? `将打开 ${protocol.toUpperCase()} ${port}` : '输入端口后显示操作预览'}</div>
              <button className="w-full rounded-xl bg-cyan-400 px-4 py-3 font-semibold text-slate-950 hover:bg-cyan-300 disabled:opacity-60" disabled={loading}>{t.openPort}</button>
            </form>
          </aside>
        </div>
      ) : null}
      {closing && <ConfirmDialog rule={closing} t={t} onCancel={() => setClosing(null)} onConfirm={() => closePort(closing)} />}
    </main>
  );
}

function Summary({ state, t }: { state: FirewallState; t: Record<string, string> }) {
  const items = [
    ['OS', state.osType],
    ['Backend', state.backend],
    ['Running', state.serviceRunning ? 'Yes' : 'No'],
    ['Enabled', state.serviceEnabled ? 'Yes' : 'No'],
    ['Policy', state.defaultIncomingPolicy],
    ['Loaded', new Date(state.loadedAt).toLocaleString()],
  ];
  return (
    <section className="rounded-3xl border border-white/10 bg-white/10 p-6">
      <h2 className="mb-4 text-xl font-semibold">{t.systemSummary}</h2>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {items.map(([label, value]) => (
          <div className="rounded-2xl bg-slate-950 p-4" key={label}>
            <div className="text-sm text-slate-400">{label}</div>
            <div className="mt-1 font-semibold">{value}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

function PortsTable({ ports, onClose, t }: { ports: PortRule[]; onClose: (rule: PortRule) => void; t: Record<string, string> }) {
  return (
    <section className="rounded-3xl border border-white/10 bg-white/10 p-6">
      <h2 className="mb-4 text-xl font-semibold">{t.openPorts}</h2>
      {ports.length === 0 ? <p className="text-slate-300">{t.noPorts}</p> : null}
      <div className="hidden overflow-hidden rounded-2xl border border-white/10 md:block">
        <table className="w-full text-left text-sm">
          <thead className="bg-white/10 text-slate-300">
            <tr><th className="p-3">{t.port}</th><th className="p-3">{t.protocol}</th><th className="p-3">{t.source}</th><th className="p-3">{t.description}</th><th className="p-3">{t.actions}</th></tr>
          </thead>
          <tbody>
            {ports.map((rule) => (
              <tr className="border-t border-white/10" key={`${rule.protocol}-${rule.port}`}>
                <td className="p-3 font-semibold">{rule.port}</td><td className="p-3 uppercase">{rule.protocol}</td><td className="p-3">{rule.source ?? 'Any'}</td><td className="p-3">{rule.description ?? '-'}</td><td className="p-3"><button onClick={() => onClose(rule)} className="rounded-lg border border-red-300/40 px-3 py-1 text-red-100 hover:bg-red-500/20">{t.close}</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="space-y-3 md:hidden">
        {ports.map((rule) => <div className="rounded-2xl bg-slate-950 p-4" key={`${rule.protocol}-${rule.port}`}><div className="text-lg font-semibold">{rule.protocol.toUpperCase()} {rule.port}</div><div className="text-sm text-slate-400">{rule.source ?? 'Any'}</div><button onClick={() => onClose(rule)} className="mt-3 rounded-lg border border-red-300/40 px-3 py-1 text-red-100">{t.close}</button></div>)}
      </div>
    </section>
  );
}

function ConfirmDialog({ rule, t, onCancel, onConfirm }: { rule: PortRule; t: Record<string, string>; onCancel: () => void; onConfirm: () => void }) {
  return (
    <div className="fixed inset-0 grid place-items-center bg-black/70 p-4">
      <div className="max-w-md rounded-3xl border border-white/10 bg-slate-900 p-6 shadow-2xl">
        <h2 className="mb-2 text-xl font-semibold">{t.confirmClose}</h2>
        <p className="mb-6 text-slate-300">{t.confirmCloseBody} ({rule.protocol.toUpperCase()} {rule.port})</p>
        <div className="flex justify-end gap-3"><button onClick={onCancel} className="rounded-xl border border-white/10 px-4 py-2">{t.cancel}</button><button onClick={onConfirm} className="rounded-xl bg-red-400 px-4 py-2 font-semibold text-slate-950">{t.confirm}</button></div>
      </div>
    </div>
  );
}

function LocaleSwitch({ locale, setLocale }: { locale: Locale; setLocale: (locale: Locale) => void }) {
  return <select className="rounded-xl border border-white/10 bg-slate-950 px-3 py-2" value={locale} onChange={(e) => setLocale(e.target.value as Locale)}><option value="zh-CN">中文</option><option value="en-US">English</option></select>;
}

function FullPageMessage({ text }: { text: string }) {
  return <div className="grid min-h-screen place-items-center text-slate-300">{text}</div>;
}

createRoot(document.getElementById('root')!).render(<App />);
