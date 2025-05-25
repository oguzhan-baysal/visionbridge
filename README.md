# VisionBridge

## 1. Proje Tanımı
VisionBridge, backend'den alınan YAML/JSON konfigürasyonları ile web sayfalarında dinamik DOM manipülasyonu yapabilen, modüler ve genişletilebilir bir sistemdir. Amaç, frontend koduna dokunmadan içerik ve layout değişikliklerini merkezi olarak yönetebilmektir.

---

## 2. Kurulum

### Backend (Go)
1. Go kurulu olmalı: https://go.dev/dl/
2. Terminalde backend klasörüne girin:
   ```sh
   cd backend
   go run main.go
   ```
3. Sunucu `localhost:8080` üzerinde çalışır.

### Frontend
1. `frontend/visionbridge.js` dosyasını projenize ekleyin.
2. HTML sayfanıza ekleyin:
   ```html
   <script src="frontend/visionbridge.js"></script>
   ```
3. Demo için `frontend/demo.html` dosyasını açabilirsiniz.

---

## 3. API Dökümantasyonu

### Konfigürasyon CRUD
- `GET    /api/configuration/all`   → Tüm konfigürasyonları getir
- `GET    /api/configuration/{id}`  → Tek konfigürasyon getir
- `POST   /api/configuration`       → Yeni konfigürasyon ekle
- `PUT    /api/configuration/{id}`  → Konfigürasyon güncelle
- `DELETE /api/configuration/{id}`  → Konfigürasyon sil

### Spesifik Konfig
- `GET    /api/specific?id=...&host=...&url=...` → Query ile uygun konfig
- `GET    /api/specific/{id}`                   → ID ile spesifik konfig
- `POST   /api/specific`                        → Yeni spesifik konfig
- `PUT    /api/specific/{id}`                   → Spesifik konfig güncelle
- `DELETE /api/specific/{id}`                   → Spesifik konfig sil

#### Örnek İstek
```sh
curl http://localhost:8080/api/configuration/all
```

---

## 4. Konfigürasyon Formatı

### Temel YAML Örneği
```yaml
id: demo
actions:
  - type: remove
    selector: ".ad-banner"
  - type: replace
    selector: "#old-header"
    newElement: "<header id='new-header'>Yeni Header</header>"
  - type: insert
    position: "after"
    target: "body"
    element: "<footer class='footer'>Footer İçeriği (VisionBridge)</footer>"
  - type: alter
    oldValue: "Machine Learning"
    newValue: "Yapay Zeka"
```

### Koşullu ve Dinamik Alanlar
```yaml
- type: remove
  selector: ".ad-banner"
  condition:
    url: "/frontend/demo.html"
    isLoggedIn: true
    queryParam:
      ref: "google"
    localStorage:
      userType: "admin"
    cookie:
      session: "active"
```

### Datasource ile Host/URL Bazlı Seçim
```yaml
datasource:
  hosts:
    "localhost": demo.yaml
  urls:
    "/frontend/demo.html": demo.yaml
```

---

## 5. Kullanım Senaryoları
- **Temel Kullanım:** Sayfa yüklendiğinde uygun konfigürasyon çekilir ve aksiyonlar uygulanır.
- **A/B Test:** Farklı host/url için farklı konfigler tanımlanabilir.
- **User Context:** Kullanıcıya özel (login, localStorage, cookie, query param) koşullu aksiyonlar.
- **Analytics Dashboard:** Sağ alt köşede açılır/kapanır panel ile canlı istatistikler.

---

## 6. Gelişmiş Özellikler
- **Çakışma Yönetimi:** Aynı selector/type için en yüksek priority'li aksiyon uygulanır.
- **Conditional Logic:** Her aksiyona koşul eklenebilir.
- **Monitoring:** Backend'de tüm API istekleri loglanır.
- **Analytics:** Frontend'de aksiyon sayaçları ve son 10 aksiyon panelde gösterilir.

---

## 7. Test ve Geliştirici Notları
- Backend'de unit ve integration testler mevcut:
  ```sh
  cd backend
  go test -v
  ```
- Genişletmek için:
  - Yeni aksiyon tipleri ekleyebilirsiniz (actions objesine yeni fonksiyon ekleyin).
  - Koşul tiplerini frontend'de `checkCondition` fonksiyonuna ekleyin.
  - Backend validasyonunu genişletmek için ilgili fonksiyonları güncelleyin.

---

## 8. Katkı ve Lisans
- Katkı yapmak için PR gönderebilirsiniz.
- MIT Lisansı ile açık kaynak. 