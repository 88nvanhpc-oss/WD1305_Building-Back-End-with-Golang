const { useEffect, useMemo, useState } = React;
const { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } = window.Recharts;
const API = localStorage.getItem('finance_api') || 'http://localhost:8080';

const money = value => new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND', maximumFractionDigits: 0 }).format(value || 0);
const dateLabel = value => new Intl.DateTimeFormat('vi-VN', { dateStyle: 'medium' }).format(new Date(value));

async function request(path, options = {}) {
  const response = await fetch(API + path, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}), ...(localStorage.getItem('finance_token') ? { Authorization: `Bearer ${localStorage.getItem('finance_token')}` } : {}) }
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || 'Có lỗi xảy ra');
  return data;
}

function Logo() { return <div className="brand"><span className="brand-mark">↗</span><span>finora</span></div>; }

function AuthPage({ onLogin }) {
  const [mode, setMode] = useState('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const submit = async () => {
    setError(''); setBusy(true);
    try {
      const data = await request(mode === 'login' ? '/login' : '/register', { method: 'POST', body: JSON.stringify({ email, password, name }) });
      localStorage.setItem('finance_token', data.token); onLogin(data.user);
    } catch (err) { setError(err.message); } finally { setBusy(false); }
  };
  return <main className="auth-shell">
    <section className="auth-visual"><Logo /><div className="visual-copy"><span className="eyebrow">YOUR MONEY, YOUR MOMENT</span><h1>Nhẹ đầu hơn,<br /><em>giàu có hơn.</em></h1><p>Một góc nhỏ để bạn hiểu rõ dòng tiền và sống theo những điều thật sự quan trọng.</p></div><div className="quote">“Kế hoạch tài chính tốt là một hình thức tự chăm sóc bản thân.”</div></section>
    <section className="auth-panel"><div className="auth-box"><Logo /><div className="auth-heading"><h2>{mode === 'login' ? 'Chào mừng trở lại' : 'Tạo tài khoản mới'}</h2><p>{mode === 'login' ? 'Đăng nhập để tiếp tục quản lý tài chính.' : 'Bắt đầu hành trình tài chính sáng rõ hơn.'}</p></div>
      {mode === 'register' && <label>Họ và tên<input value={name} onChange={e => setName(e.target.value)} placeholder="Nguyễn Văn An" /></label>}
      <label>Email<input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="ban@email.com" /></label>
      <label>Mật khẩu<input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Tối thiểu 6 ký tự" /></label>
      {error && <div className="error">{error}</div>}
      <button className="primary full" onClick={submit} disabled={busy}>{busy ? 'Đang xử lý...' : mode === 'login' ? 'Đăng nhập' : 'Tạo tài khoản'}</button>
      <div className="or"><span>hoặc</span></div>
      <button className="google full" onClick={() => { window.location.href = `${API}/auth/google`; }}><span className="google-g">G</span> Tiếp tục với Google</button>
      <p className="switch">{mode === 'login' ? 'Chưa có tài khoản?' : 'Đã có tài khoản?'} <button onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError(''); }}>{mode === 'login' ? 'Đăng ký ngay' : 'Đăng nhập'}</button></p>
    </div></section>
  </main>;
}

function App() {
  const [user, setUser] = useState(() => { try { return JSON.parse(localStorage.getItem('finance_user')) || null; } catch { return null; } });
  const [tab, setTab] = useState('overview');
  const [transactions, setTransactions] = useState([]);
  const [report, setReport] = useState({ income: 0, expense: 0, balance: 0 });
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState('');
  useEffect(() => {
    const params = new URLSearchParams(location.search); const token = params.get('token');
    if (token) { localStorage.setItem('finance_token', token); history.replaceState({}, '', location.pathname); request('/me').then(setUser).catch(() => localStorage.removeItem('finance_token')); }
  }, []);
  const loadData = async () => { setLoading(true); try { const [items, summary] = await Promise.all([request('/transactions'), request('/reports')]); setTransactions(items); setReport(summary); } catch (err) { if (err.message.includes('token')) logout(); setToast(err.message); } finally { setLoading(false); } };
  useEffect(() => { if (user) { localStorage.setItem('finance_user', JSON.stringify(user)); loadData(); } }, [user]);
  const logout = () => { localStorage.removeItem('finance_token'); localStorage.removeItem('finance_user'); setUser(null); };
  if (!user || !localStorage.getItem('finance_token')) return <AuthPage onLogin={setUser} />;
  return <div className="app-shell"><aside><Logo /><div className="profile"><div className="avatar">{(user.name || user.email).slice(0, 1).toUpperCase()}</div><div><strong>{user.name || 'Bạn'}</strong><span>{user.email}</span></div></div><nav><button className={tab === 'overview' ? 'active' : ''} onClick={() => setTab('overview')}><span>▦</span>Tổng quan</button><button className={tab === 'transactions' ? 'active' : ''} onClick={() => setTab('transactions')}><span>↔</span>Giao dịch</button><button className={tab === 'reports' ? 'active' : ''} onClick={() => setTab('reports')}><span>◒</span>Báo cáo</button></nav><button className="logout" onClick={logout}>↪ <span>Đăng xuất</span></button></aside><main className="content"><header><div><span className="eyebrow">THỨ HAI, {new Intl.DateTimeFormat('vi-VN', { day: '2-digit', month: 'long' }).format(new Date()).toUpperCase()}</span><h1>{tab === 'overview' ? 'Tổng quan của bạn' : tab === 'transactions' ? 'Giao dịch' : 'Báo cáo tài chính'}</h1></div><button className="icon-button" onClick={loadData} title="Làm mới">↻</button></header>{toast && <div className="toast" onClick={() => setToast('')}>{toast} ×</div>}{tab === 'overview' && <Overview report={report} transactions={transactions} onView={() => setTab('transactions')} />}{tab === 'transactions' && <Transactions transactions={transactions} reload={loadData} setToast={setToast} />}{tab === 'reports' && <Reports report={report} transactions={transactions} />}{loading && <div className="loading">Đang tải dữ liệu...</div>}</main></div>;
}

function Overview({ report, transactions, onView }) { return <div className="page"><div className="balance-card"><div><span className="muted">Số dư hiện tại</span><strong>{money(report.balance)}</strong><span className={report.balance >= 0 ? 'trend positive' : 'trend negative'}>{report.balance >= 0 ? '↑' : '↓'} So với tổng thu chi</span></div><div className="balance-orbit">₫</div></div><div className="stats-grid"><Stat label="Tổng thu nhập" value={report.income} tone="income" icon="↗" /><Stat label="Tổng chi tiêu" value={report.expense} tone="expense" icon="↘" /><Stat label="Số giao dịch" value={transactions.length} tone="neutral" icon="#" /></div><section className="section-head"><div><span className="eyebrow">DÒNG TIỀN</span><h2>Giao dịch gần đây</h2></div><button className="text-button" onClick={onView}>Xem tất cả →</button></section><TransactionTable items={transactions.slice(0, 5)} /></div>; }
function Stat({ label, value, tone, icon }) { return <div className="stat"><div className={`stat-icon ${tone}`}>{icon}</div><div><span>{label}</span><strong>{tone === 'neutral' ? value : money(value)}</strong></div></div>; }

function Transactions({ transactions, reload, setToast }) { const [amount, setAmount] = useState(''); const [description, setDescription] = useState(''); const [type, setType] = useState('expense'); const add = async () => { try { await request('/transactions', { method: 'POST', body: JSON.stringify({ amount: Number(amount), description, type }) }); setAmount(''); setDescription(''); setToast('Đã thêm giao dịch'); reload(); } catch (err) { setToast(err.message); } }; const remove = async id => { try { await request(`/transactions/${id}`, { method: 'DELETE' }); reload(); } catch (err) { setToast(err.message); } }; return <div className="page"><section className="transaction-layout"><div className="panel add-panel"><div className="section-head compact"><div><span className="eyebrow">GHI NHẬN</span><h2>Thêm giao dịch</h2></div><span className="plus">+</span></div><label>Số tiền<input type="number" min="1" value={amount} onChange={e => setAmount(e.target.value)} placeholder="0" /></label><label>Mô tả<input value={description} onChange={e => setDescription(e.target.value)} placeholder="Ví dụ: Tiền ăn trưa" /></label><label>Loại giao dịch<select value={type} onChange={e => setType(e.target.value)}><option value="expense">Chi tiêu</option><option value="income">Thu nhập</option></select></label><button className="primary full" onClick={add}>Lưu giao dịch</button></div><div className="panel list-panel"><div className="section-head compact"><div><span className="eyebrow">LỊCH SỬ</span><h2>Tất cả giao dịch</h2></div><span className="count">{transactions.length} giao dịch</span></div><TransactionTable items={transactions} onDelete={remove} /></div></section></div>; }

function TransactionTable({ items, onDelete }) { if (!items.length) return <div className="empty">Chưa có giao dịch nào.<br /><span>Thêm giao dịch đầu tiên để bắt đầu theo dõi.</span></div>; return <div className="transaction-list">{items.map(item => <div className="transaction" key={item.id}><div className={`tx-icon ${item.type}`}>{item.type === 'income' ? '↗' : '↘'}</div><div className="tx-main"><strong>{item.description}</strong><span>{dateLabel(item.created_at)}</span></div><strong className={item.type === 'income' ? 'amount income-text' : 'amount expense-text'}>{item.type === 'income' ? '+' : '-'}{money(item.amount)}</strong>{onDelete && <button className="delete" onClick={() => onDelete(item.id)} title="Xóa">×</button>}</div>)}</div>; }

function Reports({ report, transactions }) { const data = [{ name: 'Thu nhập', value: report.income, color: '#32b48b' }, { name: 'Chi tiêu', value: report.expense, color: '#ee7d60' }]; return <div className="page"><div className="report-grid"><section className="panel chart-panel"><div className="section-head compact"><div><span className="eyebrow">PHÂN TÍCH</span><h2>Thu nhập & chi tiêu</h2></div></div><div className="chart-wrap">{report.income + report.expense > 0 ? <ResponsiveContainer width="100%" height="100%"><PieChart><Pie data={data} innerRadius={75} outerRadius={112} paddingAngle={4} dataKey="value">{data.map(entry => <Cell key={entry.name} fill={entry.color} />)}</Pie><Tooltip formatter={value => money(value)} /></PieChart></ResponsiveContainer> : <div className="empty">Chưa đủ dữ liệu để hiển thị biểu đồ.</div>}<div className="chart-center"><strong>{money(report.balance)}</strong><span>Số dư ròng</span></div></div><div className="legend">{data.map(entry => <div key={entry.name}><span style={{ background: entry.color }}></span>{entry.name}<strong>{money(entry.value)}</strong></div>)}</div></section><section className="panel insight-panel"><span className="eyebrow">TỔNG KẾT</span><h2>Thói quen tài chính</h2><p>{report.income === 0 && report.expense === 0 ? 'Hãy thêm các giao dịch đầu tiên để nhận tổng kết.' : report.balance >= 0 ? 'Bạn đang duy trì dòng tiền dương. Tiếp tục giữ thói quen theo dõi đều đặn.' : 'Chi tiêu hiện cao hơn thu nhập. Hãy xem lại các khoản gần đây để cân bằng ngân sách.'}</p><div className="mini-metrics"><div><span>Thu nhập</span><strong>{money(report.income)}</strong></div><div><span>Chi tiêu</span><strong>{money(report.expense)}</strong></div></div><div className="progress"><span style={{ width: `${report.income ? Math.min(100, report.expense / report.income * 100) : 0}%` }}></span></div><small>Mức chi tiêu so với thu nhập</small></section></div></div>; }

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
