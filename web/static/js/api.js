// ═══════════════════════════════════════════════════════════════════════════════
// api.js — Client-side API layer with PASETO token management & Google OAuth
// ═══════════════════════════════════════════════════════════════════════════════

const API_BASE = '/v1';

const Api = {
    // ─── Token Management ─────────────────────────────────────────────────
    isAuthenticated: () => !!localStorage.getItem('access_token'),

    getToken: () => localStorage.getItem('access_token'),

    getRefreshToken: () => localStorage.getItem('refresh_token'),

    setTokens: (access, refresh) => {
        localStorage.setItem('access_token', access);
        if (refresh) localStorage.setItem('refresh_token', refresh);
    },

    clearTokens: () => {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        localStorage.removeItem('user_info');
        sessionStorage.removeItem('oauth_state');
    },

    getUserInfo: () => {
        try {
            return JSON.parse(localStorage.getItem('user_info'));
        } catch {
            return null;
        }
    },

    // ─── Core HTTP Request ────────────────────────────────────────────────
    request: async (endpoint, method = 'GET', body = null, _retried = false) => {
        const headers = { 'Content-Type': 'application/json' };
        const token = Api.getToken();
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const options = { method, headers };
        if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
            options.body = JSON.stringify(body);
        }

        const res = await fetch(`${API_BASE}${endpoint}`, options);

        // Handle 401 — attempt token refresh once
        if (res.status === 401 && !_retried) {
            const refreshed = await Api.tryRefreshToken();
            if (refreshed) {
                return Api.request(endpoint, method, body, true);
            }
            Api.clearTokens();
            window.location.href = '/';
            throw new Error('Session expired. Please log in again.');
        }

        let data;
        try {
            const text = await res.text();
            data = text ? JSON.parse(text) : {};
        } catch (e) {
            data = {};
        }

        if (!res.ok) {
            const msg = data.message || data.detail || data.error || `Request failed (${res.status})`;
            throw new Error(msg);
        }

        return data;
    },

    // ─── Token Refresh ────────────────────────────────────────────────────
    tryRefreshToken: async () => {
        const refreshToken = Api.getRefreshToken();
        if (!refreshToken) return false;

        try {
            const res = await fetch(`${API_BASE}/auth/renew`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ refresh_token: refreshToken }),
            });

            if (!res.ok) return false;

            const data = await res.json();
            if (data && data.access_token) {
                Api.setTokens(data.access_token, null);
                return true;
            }
            return false;
        } catch {
            return false;
        }
    },

    // ─── Logout ───────────────────────────────────────────────────────────
    logout: async () => {
        try {
            await Api.request('/auth/logout', 'POST', {});
        } catch {
            // Even if the API call fails, clear local tokens
        }
        Api.clearTokens();
        window.location.href = '/';
    },
};

// ═══════════════════════════════════════════════════════════════════════════════
// NAV SETUP — runs on every page
// ═══════════════════════════════════════════════════════════════════════════════

function setupNav() {
    const navActions = document.getElementById('nav-actions');
    if (!navActions) return;

    if (Api.isAuthenticated()) {
        const user = Api.getUserInfo();
        const initial = user?.full_name?.[0]?.toUpperCase() || user?.email?.[0]?.toUpperCase() || '?';
        const name = user?.full_name || user?.email || 'User';

        navActions.innerHTML = `
            <div class="nav-user">
                <button class="nav-user-info" onclick="if(window.openProfileModal) window.openProfileModal()" style="cursor:pointer; background:transparent; border:none; color:inherit; text-align:left; font-family:inherit;">
                    <div class="nav-avatar">${escapeHtml(initial)}</div>
                    <span style="font-weight: 700;">${escapeHtml(name)} <i class="ph-bold ph-caret-down" style="font-size: 0.8rem; opacity: 0.6;"></i></span>
                </button>
                <button onclick="Api.logout()" class="btn btn-danger btn-sm">
                    <i class="ph-bold ph-sign-out"></i> Logout
                </button>
            </div>`;
    }
}

document.addEventListener('DOMContentLoaded', setupNav);

// ═══════════════════════════════════════════════════════════════════════════════
// TOAST NOTIFICATIONS
// ═══════════════════════════════════════════════════════════════════════════════

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    const icons = { success: 'ph-check-circle', error: 'ph-warning-circle', info: 'ph-info' };
    toast.innerHTML = `<i class="ph-bold ${icons[type] || icons.info}"></i> ${escapeHtml(message)}`;
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(30px)';
        toast.style.transition = 'all 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 3500);
}

// ═══════════════════════════════════════════════════════════════════════════════
// UTILITY
// ═══════════════════════════════════════════════════════════════════════════════

function escapeHtml(unsafe) {
    return (unsafe || '').toString()
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}
