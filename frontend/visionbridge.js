// VisionBridge Dinamik Sayfa Değiştirici (Vanilla JS)
(function () {
  const API_URL = "http://localhost:8080/api/configuration/all";

  // Yardımcı: DOM aksiyonları
  const actions = {
    remove: ({ selector }) => {
      document.querySelectorAll(selector).forEach((el) => el.remove());
    },
    replace: ({ selector, newElement }) => {
      document.querySelectorAll(selector).forEach((el) => {
        const temp = document.createElement("div");
        temp.innerHTML = newElement;
        const newEl = temp.firstElementChild;
        if (newEl) el.replaceWith(newEl);
      });
    },
    insert: ({ position, target, element }) => {
      document.querySelectorAll(target).forEach((el) => {
        const temp = document.createElement("div");
        temp.innerHTML = element;
        const newEl = temp.firstElementChild;
        if (!newEl) return;
        switch (position) {
          case "before":
            el.parentNode.insertBefore(newEl, el);
            break;
          case "after":
            el.parentNode.insertBefore(newEl, el.nextSibling);
            break;
          case "prepend":
            el.insertBefore(newEl, el.firstChild);
            break;
          case "append":
          default:
            el.appendChild(newEl);
        }
      });
    },
    alter: ({ oldValue, newValue }) => {
      // Tüm metin düğümlerinde değiştir
      const treeWalker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      let node;
      while ((node = treeWalker.nextNode())) {
        if (node.nodeValue.includes(oldValue)) {
          node.nodeValue = node.nodeValue.replaceAll(oldValue, newValue);
        }
      }
    },
  };

  // Analytics: aksiyon sayaçları ve log
  const analytics = {
    counts: {},
    lastApplied: {},
    logs: [], // {type, selector, time}
  };

  // Çakışma yönetimi: aynı selector ve type için en yüksek priority'li aksiyonu uygula
  function resolveConflicts(actionsArr) {
    const map = new Map();
    actionsArr.forEach((action, idx) => {
      const key = action.type + '|' + (action.selector || action.target || action.oldValue || idx);
      if (!map.has(key)) {
        map.set(key, action);
      } else {
        const prev = map.get(key);
        const prevPriority = typeof prev.priority === 'number' ? prev.priority : -Infinity;
        const currPriority = typeof action.priority === 'number' ? action.priority : -Infinity;
        if (currPriority > prevPriority) {
          map.set(key, action);
        } else if (currPriority === prevPriority) {
          // Priority eşitse, sonuncusu kazanır
          map.set(key, action);
        }
      }
    });
    return Array.from(map.values());
  }

  // Uygun konfigürasyonu seç (host > url > ilk)
  function selectConfig(configs) {
    const host = window.location.hostname;
    const path = window.location.pathname;
    // 1. datasource varsa, host/url eşleşmesi ara
    for (const config of configs) {
      if (config.datasource) {
        // Host bazlı
        if (config.datasource.hosts && config.datasource.hosts[host]) {
          return config;
        }
        // URL bazlı
        if (config.datasource.urls && config.datasource.urls[path]) {
          return config;
        }
      }
    }
    // 2. datasource yoksa veya eşleşme yoksa ilk konfig
    return configs[0];
  }

  // Koşullu aksiyon kontrolü (user context/dynamic values dahil)
  function checkCondition(condition) {
    if (!condition) return true;
    if (condition.url && window.location.pathname !== condition.url) return false;
    if (condition.host && window.location.hostname !== condition.host) return false;
    if (condition.userAgentIncludes && !navigator.userAgent.includes(condition.userAgentIncludes)) return false;
    // isLoggedIn: localStorage veya cookie'de isLoggedIn anahtarı true ise
    if (typeof condition.isLoggedIn !== 'undefined') {
      let logged = false;
      try {
        logged = localStorage.getItem('isLoggedIn') === 'true';
      } catch {}
      if (!logged) {
        // Cookie'de de bak
        logged = document.cookie.split(';').some(c => c.trim().startsWith('isLoggedIn=true'));
      }
      if (condition.isLoggedIn !== logged) return false;
    }
    // queryParam: { anahtar: değer }
    if (condition.queryParam) {
      const params = new URLSearchParams(window.location.search);
      for (const key in condition.queryParam) {
        if (params.get(key) !== condition.queryParam[key]) return false;
      }
    }
    // localStorage: { anahtar: değer }
    if (condition.localStorage) {
      for (const key in condition.localStorage) {
        try {
          if (localStorage.getItem(key) !== condition.localStorage[key]) return false;
        } catch { return false; }
      }
    }
    // cookie: { anahtar: değer }
    if (condition.cookie) {
      for (const key in condition.cookie) {
        const val = condition.cookie[key];
        const found = document.cookie.split(';').some(c => c.trim() === `${key}=${val}`);
        if (!found) return false;
      }
    }
    return true;
  }

  // Dashboard panelini oluştur
  function createDashboard() {
    if (document.getElementById('vb-dashboard')) return;
    const panel = document.createElement('div');
    panel.id = 'vb-dashboard';
    panel.style = `position:fixed;bottom:16px;right:16px;z-index:99999;background:#18181b;color:#fff;padding:16px 20px;border-radius:12px;box-shadow:0 2px 16px #0008;font-size:14px;min-width:260px;max-width:340px;max-height:60vh;overflow:auto;display:none;`;
    panel.innerHTML = `
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
        <b>VisionBridge Analytics</b>
        <button id="vb-close" style="background:none;border:none;color:#fff;font-size:18px;cursor:pointer;">×</button>
      </div>
      <div id="vb-analytics-content"></div>
    `;
    document.body.appendChild(panel);
    document.getElementById('vb-close').onclick = () => { panel.style.display = 'none'; };

    // Toggle butonu
    const btn = document.createElement('button');
    btn.id = 'vb-dashboard-btn';
    btn.innerText = 'Analytics';
    btn.style = 'position:fixed;bottom:16px;right:16px;z-index:99998;background:#2563eb;color:#fff;padding:8px 16px;border-radius:8px;border:none;box-shadow:0 2px 8px #0004;cursor:pointer;font-size:14px;';
    btn.onclick = () => { panel.style.display = panel.style.display === 'none' ? 'block' : 'none'; };
    document.body.appendChild(btn);
  }

  // Dashboard içeriğini güncelle
  function updateDashboard() {
    const el = document.getElementById('vb-analytics-content');
    if (!el) return;
    let html = '<b>Aksiyon Sayaçları</b><ul style="margin:8px 0 12px 0;padding-left:18px;">';
    for (const type in analytics.counts) {
      html += `<li><b>${type}</b>: ${analytics.counts[type]}</li>`;
    }
    html += '</ul>';
    html += '<b>Son 10 Aksiyon</b><ul style="margin:8px 0 0 0;padding-left:18px;">';
    analytics.logs.slice(-10).reverse().forEach(log => {
      html += `<li>${log.time} <b>${log.type}</b> <span style='color:#a3e635'>${log.selector||''}</span></li>`;
    });
    html += '</ul>';
    el.innerHTML = html;
  }

  // Konfigürasyonları uygula
  function applyConfig(config) {
    if (!config.actions) return;
    createDashboard();
    const filteredActions = resolveConflicts(config.actions);
    filteredActions.forEach((action) => {
      if (actions[action.type]) {
        // Koşul kontrolü
        if (!checkCondition(action.condition)) return;
        try {
          actions[action.type](action);
          // Analytics logla
          analytics.counts[action.type] = (analytics.counts[action.type] || 0) + 1;
          analytics.lastApplied[action.type] = new Date().toISOString();
          analytics.logs.push({
            type: action.type,
            selector: action.selector || action.target || '',
            time: new Date().toLocaleTimeString(),
          });
          if (analytics.logs.length > 100) analytics.logs = analytics.logs.slice(-100);
          updateDashboard();
        } catch (e) {
          console.warn("VisionBridge action error:", action, e);
        }
      }
    });
    // Analytics'i globalde erişilebilir yap
    window.VisionBridgeAnalytics = analytics;
    updateDashboard();
  }

  // API'den konfigürasyonları çek
  fetch(API_URL)
    .then((res) => res.json())
    .then((configs) => {
      if (!Array.isArray(configs) || configs.length === 0) return;
      const config = selectConfig(configs);
      applyConfig(config);
    })
    .catch((err) => {
      console.error("VisionBridge config fetch error:", err);
    });
})(); 