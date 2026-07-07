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

type LocaleInfo = {
  locale: Locale;
  supportedLocales: Locale[];
};

type PortRule = {
  port: string;
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

class ApiError extends Error {
  code: string;

  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

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
    portHelp: '支持单个端口、范围和序列，例如 80、8000-8010、80,443,10000-10010。',
    tlsWarning: '当前未启用 HTTPS，请仅在可信网络使用。',
    localHttp: '当前为本机 HTTP 模式。',
    invalidLogin: '用户名或密码错误',
    loading: '加载中...',
    noPorts: '当前没有由系统识别的开放端口。',
    confirmClose: '确认关闭端口',
    confirmCloseBody: '关闭该端口可能导致依赖它的服务无法从外部访问。',
    cancel: '取消',
    confirm: '确认关闭',
    yes: '是',
    no: '否',
    policyAllow: '允许',
    policyDeny: '拒绝',
    policyReject: '拒绝并返回错误',
    policyUnknown: '未知',
    sourceAny: '任意来源',
    previewEmpty: '输入端口后显示操作预览',
    previewOpen: '将打开 {protocol} {port}',
    unknownError: '操作失败，请重试。',
    INVALID_JSON: '请求数据格式无效。',
    AUTH_INVALID_CREDENTIALS: '用户名或密码错误。',
    AUTH_REQUIRED: '请先登录。',
    INTERNAL_ERROR: '服务器内部错误。',
    FIREWALL_STATE_LOAD_FAILED: '无法读取防火墙状态。',
    PORT_INVALID: '端口或协议无效。',
    PROTOCOL_INVALID: '协议必须是 TCP 或 UDP。',
    PORT_OPEN_FAILED: '打开端口失败。',
    PORT_CLOSE_FAILED: '关闭端口失败。',
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
    portHelp: 'Supports a single port, a range, or a sequence, for example 80, 8000-8010, 80,443,10000-10010.',
    tlsWarning: 'HTTPS is disabled. Use only on a trusted network.',
    localHttp: 'Running in local HTTP mode.',
    invalidLogin: 'Invalid username or password',
    loading: 'Loading...',
    noPorts: 'No recognized open ports.',
    confirmClose: 'Close port?',
    confirmCloseBody: 'Closing this port may make dependent services unavailable externally.',
    cancel: 'Cancel',
    confirm: 'Close port',
    yes: 'Yes',
    no: 'No',
    policyAllow: 'Allow',
    policyDeny: 'Deny',
    policyReject: 'Reject',
    policyUnknown: 'Unknown',
    sourceAny: 'Any',
    previewEmpty: 'Enter a port to preview the operation',
    previewOpen: 'Will open {protocol} {port}',
    unknownError: 'Operation failed. Please try again.',
    INVALID_JSON: 'Invalid request data.',
    AUTH_INVALID_CREDENTIALS: 'Invalid username or password.',
    AUTH_REQUIRED: 'Please sign in first.',
    INTERNAL_ERROR: 'Internal server error.',
    FIREWALL_STATE_LOAD_FAILED: 'Failed to load firewall state.',
    PORT_INVALID: 'Invalid port or protocol.',
    PROTOCOL_INVALID: 'Protocol must be TCP or UDP.',
    PORT_OPEN_FAILED: 'Failed to open port.',
    PORT_CLOSE_FAILED: 'Failed to close port.',
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
    const code = body?.error?.code ?? `HTTP_${response.status}`;
    throw new ApiError(code, body?.error?.message ?? code);
  }
  return response.json() as Promise<T>;
}

function localizedError(err: unknown, t: Record<string, string>): string {
  if (err instanceof ApiError) {
    return t[err.code] ?? err.message ?? t.unknownError;
  }
  return err instanceof Error ? err.message : t.unknownError;
}

function translatePolicy(policy: string, t: Record<string, string>): string {
  switch (policy.toLowerCase()) {
    case 'allow':
      return t.policyAllow;
    case 'deny':
      return t.policyDeny;
    case 'reject':
      return t.policyReject;
    default:
      return t.policyUnknown;
  }
}

function translateSource(source: string | undefined, t: Record<string, string>): string {
  return !source || source.toLowerCase() === 'any' ? t.sourceAny : source;
}

function isPortExpression(value: string): boolean {
  if (!value) return false;
  return value.split(',').every((part) => {
    const bounds = part.trim().split('-');
    if (bounds.length > 2) return false;
    const start = parsePortNumber(bounds[0]);
    const end = bounds.length === 2 ? parsePortNumber(bounds[1]) : start;
    return start !== null && end !== null && start <= end;
  });
}

function normalizePortExpression(value: string): string {
  return value.replace(/[\u3001\uFF0C]/g, ',').replace(/[\uFE63\uFF0D\u2010-\u2015]/g, '-');
}

function parsePortNumber(value: string): number | null {
  if (!/^\d+$/.test(value.trim())) return null;
  const port = Number(value);
  return Number.isInteger(port) && port >= 1 && port <= 65535 ? port : null;
}

function App() {
  const [locale, setLocale] = useState<Locale>('zh-CN');
  const [runtime, setRuntime] = useState<RuntimeInfo | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [authChecked, setAuthChecked] = useState(false);

  const t = messages[locale];

  useEffect(() => {
    api<LocaleInfo>('/api/locale').then((info) => setLocale(info.locale)).catch(() => undefined);
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
  const [username, setUsername] = useState('');
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
      setError(localizedError(err, t));
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
    const normalizedPort = normalizePortExpression(port).trim();
    if (!isPortExpression(normalizedPort)) {
      setError(t.PORT_INVALID);
      return;
    }
    setLoading(true);
    setError('');
    try {
      const result = await api<{ state: FirewallState }>('/api/firewall/ports', { method: 'POST', body: JSON.stringify({ port: normalizedPort, protocol }) });
      setState(result.state);
      setPort('');
    } catch (err) {
      setError(localizedError(err, t));
    } finally {
      setLoading(false);
    }
  }

  async function closePort(rule: PortRule) {
    setLoading(true);
    setError('');
    try {
      const result = await api<{ state: FirewallState }>(`/api/firewall/ports/${rule.protocol}/${encodeURIComponent(rule.port)}`, { method: 'DELETE' });
      setState(result.state);
      setClosing(null);
    } catch (err) {
      setError(localizedError(err, t));
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
              <input id="port" inputMode="text" placeholder="80,443,10000-10010" className="w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3 outline-none ring-cyan-400 focus:ring-2" value={port} onChange={(e) => setPort(normalizePortExpression(e.target.value))} />
              <p className="text-sm text-slate-400">{t.portHelp}</p>
              <label className="block text-sm text-slate-300" htmlFor="protocol">{t.protocol}</label>
              <select id="protocol" className="w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3" value={protocol} onChange={(e) => setProtocol(e.target.value as 'tcp' | 'udp')}>
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
              </select>
              <div className="rounded-xl bg-slate-950 p-3 text-sm text-slate-300">{port ? t.previewOpen.replace('{protocol}', protocol.toUpperCase()).replace('{port}', port) : t.previewEmpty}</div>
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
    ['Running', state.serviceRunning ? t.yes : t.no],
    ['Enabled', state.serviceEnabled ? t.yes : t.no],
    ['Policy', translatePolicy(state.defaultIncomingPolicy, t)],
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
                <td className="p-3 font-semibold">{rule.port}</td><td className="p-3 uppercase">{rule.protocol}</td><td className="p-3">{translateSource(rule.source, t)}</td><td className="p-3">{rule.description ?? '-'}</td><td className="p-3"><button onClick={() => onClose(rule)} className="rounded-lg border border-red-300/40 px-3 py-1 text-red-100 hover:bg-red-500/20">{t.close}</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="space-y-3 md:hidden">
        {ports.map((rule) => <div className="rounded-2xl bg-slate-950 p-4" key={`${rule.protocol}-${rule.port}`}><div className="text-lg font-semibold">{rule.protocol.toUpperCase()} {rule.port}</div><div className="text-sm text-slate-400">{translateSource(rule.source, t)}</div><button onClick={() => onClose(rule)} className="mt-3 rounded-lg border border-red-300/40 px-3 py-1 text-red-100">{t.close}</button></div>)}
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
  async function changeLocale(nextLocale: Locale) {
    setLocale(nextLocale);
    await api('/api/locale', { method: 'POST', body: JSON.stringify({ locale: nextLocale }) }).catch(() => undefined);
  }

  return <select className="rounded-xl border border-white/10 bg-slate-950 px-3 py-2" value={locale} onChange={(e) => changeLocale(e.target.value as Locale)}><option value="zh-CN">中文</option><option value="en-US">English</option></select>;
}

function FullPageMessage({ text }: { text: string }) {
  return <div className="grid min-h-screen place-items-center text-slate-300">{text}</div>;
}

createRoot(document.getElementById('root')!).render(<App />);
