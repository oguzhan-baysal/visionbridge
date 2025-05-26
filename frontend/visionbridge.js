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

  // Uygun konfigürasyonu seç (pages > host > url > ilk)
  function selectConfig(configs) {
    const host = window.location.hostname;
    const path = window.location.pathname;
    
    // 1. Pages bazlı konfigürasyon kontrolü
    for (const config of configs) {
      if (config.datasource && config.datasource.pages) {
        // Sayfa tipini belirle (basit heuristic)
        const pageType = detectPageType(path);
        if (pageType && config.datasource.pages[pageType]) {
          console.log(`VisionBridge: Pages config seçildi - ${pageType}`);
          return config;
        }
      }
    }
    
    // 2. Host bazlı eşleşme
    for (const config of configs) {
      if (config.datasource && config.datasource.hosts && config.datasource.hosts[host]) {
        console.log(`VisionBridge: Host config seçildi - ${host}`);
        return config;
      }
    }
    
    // 3. URL bazlı eşleşme
    for (const config of configs) {
      if (config.datasource && config.datasource.urls) {
        // Tam eşleşme
        if (config.datasource.urls[path]) {
          console.log(`VisionBridge: URL config seçildi - ${path}`);
          return config;
        }
        // Partial eşleşme (path başlangıcı)
        for (const urlPattern in config.datasource.urls) {
          if (path.startsWith(urlPattern)) {
            console.log(`VisionBridge: URL pattern config seçildi - ${urlPattern}`);
            return config;
          }
        }
      }
    }
    
    // 4. Varsayılan: ilk konfigürasyon
    console.log("VisionBridge: Varsayılan config seçildi");
    return configs[0];
  }

  // Sayfa tipini belirle (basit heuristic)
  function detectPageType(path) {
    // E-commerce patterns
    if (path.includes('/product/') || path.includes('/item/')) return 'details';
    if (path.includes('/products') || path.includes('/shop') || path.includes('/catalog')) return 'list';
    if (path.includes('/cart') || path.includes('/basket')) return 'cart';
    if (path.includes('/checkout') || path.includes('/payment')) return 'checkout';
    
    // Blog patterns
    if (path.includes('/post/') || path.includes('/article/')) return 'post';
    if (path.includes('/category/') || path.includes('/tag/')) return 'category';
    if (path.includes('/archive')) return 'archive';
    
    // General patterns
    if (path === '/' || path === '/home') return 'home';
    if (path.includes('/about')) return 'about';
    if (path.includes('/contact')) return 'contact';
    if (path.includes('/search')) return 'search';
    if (path.includes('/profile') || path.includes('/account')) return 'profile';
    
    return null;
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
    
    let html = '';
    
    // Fetch durumu
    if (analytics.lastFetch) {
      const fetch = analytics.lastFetch;
      const statusColor = fetch.success ? '#22c55e' : '#ef4444';
      const statusText = fetch.success ? 'Başarılı' : 'Başarısız';
      const source = fetch.fromCache ? ' (Cache)' : '';
      
      html += `<b>API Durumu</b><div style="margin:8px 0 12px 0;padding:8px;background:#27272a;border-radius:6px;">`;
      html += `<div style="color:${statusColor};">● ${statusText}${source}</div>`;
      if (fetch.attempt) html += `<div style="font-size:12px;color:#a1a1aa;">Deneme: ${fetch.attempt}</div>`;
      if (fetch.configCount) html += `<div style="font-size:12px;color:#a1a1aa;">Konfigürasyon: ${fetch.configCount}</div>`;
      if (fetch.selectedConfig) {
        html += `<div style="font-size:12px;color:#a3e635;">Seçilen: ${fetch.selectedConfig}</div>`;
      }
      if (fetch.pageType) {
        html += `<div style="font-size:12px;color:#60a5fa;">Sayfa Tipi: ${fetch.pageType}</div>`;
      }
      if (fetch.error) html += `<div style="font-size:12px;color:#fca5a5;">${fetch.error}</div>`;
      html += `<div style="font-size:12px;color:#a1a1aa;">${new Date(fetch.timestamp).toLocaleString()}</div>`;
      html += '</div>';
    }
    
    html += '<b>Aksiyon Sayaçları</b><ul style="margin:8px 0 12px 0;padding-left:18px;">';
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

  // Retry mechanism ile API çağrısı
  async function fetchConfigWithRetry(url, maxRetries = 3, delay = 1000) {
    let lastError;
    
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        console.log(`VisionBridge: API çağrısı deneme ${attempt}/${maxRetries}`);
        
        const response = await fetch(url, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
          // Timeout ekle (10 saniye)
          signal: AbortSignal.timeout ? AbortSignal.timeout(10000) : undefined
        });
        
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const configs = await response.json();
        
        if (!Array.isArray(configs) || configs.length === 0) {
          console.warn("VisionBridge: Boş konfigürasyon listesi alındı");
          return;
        }
        
        console.log(`VisionBridge: ${configs.length} konfigürasyon başarıyla alındı`);
        const config = selectConfig(configs);
        
        // Sayfa tipi ve seçilen config bilgisini al
        const pageType = detectPageType(window.location.pathname);
        const configName = config.name || config.id || 'Bilinmeyen';
        
        applyConfig(config);
        
        // Analytics'e başarılı fetch bilgisi ekle
        analytics.lastFetch = {
          success: true,
          attempt: attempt,
          timestamp: new Date().toISOString(),
          configCount: configs.length,
          selectedConfig: configName,
          pageType: pageType
        };
        
        return; // Başarılı, döngüden çık
        
      } catch (error) {
        lastError = error;
        console.warn(`VisionBridge: Deneme ${attempt} başarısız:`, error.message);
        
        // Analytics'e hata bilgisi ekle
        analytics.lastFetch = {
          success: false,
          attempt: attempt,
          error: error.message,
          timestamp: new Date().toISOString()
        };
        
        // Son deneme değilse bekle
        if (attempt < maxRetries) {
          const waitTime = delay * Math.pow(2, attempt - 1); // Exponential backoff
          console.log(`VisionBridge: ${waitTime}ms bekleyip tekrar denenecek...`);
          await new Promise(resolve => setTimeout(resolve, waitTime));
        }
      }
    }
    
    // Tüm denemeler başarısız
    console.error(`VisionBridge: ${maxRetries} deneme sonrası başarısız:`, lastError);
    
    // Fallback: localStorage'dan cached config kullan
    try {
      const cachedConfig = localStorage.getItem('visionbridge-cache');
      if (cachedConfig) {
        console.log("VisionBridge: Cached konfigürasyon kullanılıyor");
        const configs = JSON.parse(cachedConfig);
        const config = selectConfig(configs);
        applyConfig(config);
        
        analytics.lastFetch = {
          success: true,
          fromCache: true,
          timestamp: new Date().toISOString()
        };
      }
    } catch (cacheError) {
      console.warn("VisionBridge: Cache'den okuma başarısız:", cacheError);
    }
    
    // Custom event dispatch et
    window.dispatchEvent(new CustomEvent('visionbridge:fetch-failed', {
      detail: { error: lastError, attempts: maxRetries }
    }));
  }

  // Konfigürasyonları cache'le
  function cacheConfig(configs) {
    try {
      localStorage.setItem('visionbridge-cache', JSON.stringify(configs));
      localStorage.setItem('visionbridge-cache-time', Date.now().toString());
    } catch (error) {
      console.warn("VisionBridge: Cache yazma başarısız:", error);
    }
  }

  // Cache'i kontrol et (1 saat geçerliliği)
  function getCachedConfig() {
    try {
      const cacheTime = localStorage.getItem('visionbridge-cache-time');
      const oneHour = 60 * 60 * 1000;
      
      if (cacheTime && (Date.now() - parseInt(cacheTime)) < oneHour) {
        const cached = localStorage.getItem('visionbridge-cache');
        if (cached) {
          return JSON.parse(cached);
        }
      }
    } catch (error) {
      console.warn("VisionBridge: Cache okuma başarısız:", error);
    }
    return null;
  }

  // Ana fetch fonksiyonu
  async function initVisionBridge() {
    // Önce cache'i kontrol et
    const cachedConfigs = getCachedConfig();
    if (cachedConfigs) {
      console.log("VisionBridge: Cache'den konfigürasyon kullanılıyor");
      const config = selectConfig(cachedConfigs);
      applyConfig(config);
      
      analytics.lastFetch = {
        success: true,
        fromCache: true,
        timestamp: new Date().toISOString()
      };
    }
    
    // Her durumda fresh data almaya çalış
    try {
      await fetchConfigWithRetry(API_URL);
      
      // Başarılı fetch sonrası cache'i güncelle
      const response = await fetch(API_URL);
      if (response.ok) {
        const configs = await response.json();
        cacheConfig(configs);
      }
    } catch (error) {
      // Hata durumunda cache kullanıldı, ek bir şey yapmaya gerek yok
    }
  }

  // VisionBridge'i başlat
  initVisionBridge();
})(); 