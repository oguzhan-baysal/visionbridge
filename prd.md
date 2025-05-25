PRD.md - VisionBridge Dinamik Sayfa Değiştirici
1. Proje Genel Bakışı
1.1 Amaç
Web sayfalarını dinamik olarak değiştirmek için konfigürasyon tabanlı bir sistem geliştirmek. Sistem, backend'den alınan YAML/JSON yapılandırmaları kullanarak DOM manipülasyonu yapar.
1.2 Hedef Kullanıcılar

Frontend geliştiriciler (kütüphaneyi entegre edenler)
İçerik yöneticileri (konfigürasyonları düzenleyenler)
Web sitesi yöneticileri

2. Fonksiyonel Gereksinimler
2.1 Frontend Kütüphanesi
2.1.1 Konfigürasyon Yönetimi

Gereksinim: Backend'den YAML/JSON konfigürasyonlarını alabilmeli
Detay:

RESTful API çağrıları
Hata yönetimi ve retry mekanizması
Multiple konfigürasyon desteği



2.1.2 DOM Manipülasyon Aksiyonları
Remove Aksiyonu

Gereksinim: Belirtilen DOM elementlerini kaldırabilmeli
Desteklenen seçiciler: CSS selector (class, id, tag)
Örnek: .ad-banner sınıfındaki tüm elementleri kaldır

Replace Aksiyonu

Gereksinim: Mevcut elementleri yenisiyle değiştirebilmeli
Detay: HTML string olarak yeni element kabul eder
Örnek: #old-header id'li elementi yeni header ile değiştir

Insert Aksiyonu

Gereksinim: Belirtilen pozisyona yeni element ekleyebilmeli
Desteklenen pozisyonlar: before, after, prepend, append
Hedef: CSS selector ile belirlenen element

Alter Aksiyonu

Gereksinim: Metin içeriğini değiştirebilmeli
Detay: Belirtilen metni yenisiyle değiştir
Örnek: "Machine Learning" → "AI"

2.1.3 Çakışma Yönetimi

Gereksinim: Birden fazla konfigürasyonda çakışma durumunda öncelik mantığı
Stratejiler:

Son gelen kazanır
Öncelik puanı sistemi
Aksiyon tipine göre öncelik



2.1.4 Sayfa Spesifik Konfigürasyon

Gereksinim: URL, sayfa türü veya host bazında farklı konfigürasyonlar
Detay:

Mevcut URL'yi analiz etme
Host bilgisini çıkarma
Sayfa türünü belirleme (opsiyonel)



2.2 Backend API Servisi
2.2.1 Konfigürasyon CRUD İşlemleri
GET /api/configuration/{id} - Tek konfigürasyon getir
GET /api/configuration/all - Tüm konfigürasyonları getir
POST /api/configuration - Yeni konfigürasyon ekle
PUT /api/configuration/{id} - Konfigürasyon güncelle
DELETE /api/configuration/{id} - Konfigürasyon sil
2.2.2 Spesifik Konfigürasyon Yönetimi
GET /api/specific - Query parameter ile spesifik konfigürasyon
GET /api/specific/{id} - ID ile spesifik konfigürasyon
POST /api/specific - Yeni spesifik konfigürasyon
PUT /api/specific/{id} - Spesifik konfigürasyon güncelle
DELETE /api/specific/{id} - Spesifik konfigürasyon sil
2.2.3 Validasyon ve Hata Yönetimi

YAML/JSON format validasyonu
Aksiyon tipi kontrolü
CSS selector validasyonu
HTML content sanitization

3. Teknik Gereksinimler
3.1 Frontend

Dil: JavaScript (ES6+)
Bağımlılık: Minimal (vanilla JS tercih edilir)
Browser Desteği: Modern browserlar
Yapı: Modüler ve genişletilebilir

3.2 Backend

Dil: Go veya Java
Framework: Herhangi bir framework kullanılabilir
Database: İsteğe bağlı (başlangıçta file-based)
API: RESTful

3.3 Data Format
yaml# Konfigürasyon Örneği
actions:
  - type: remove
    selector: ".ad-banner"
  - type: replace
    selector: "#old-header"
    newElement: "<header id='new-header'>New Header</header>"
  - type: insert
    position: "after"
    target: "body"
    element: "<footer>Footer Content</footer>"
  - type: alter
    oldValue: "Machine Learning"
    newValue: "AI"
yaml# Spesifik Konfigürasyon Örneği
datasource:
  pages:
    list: A.yaml
    details: B.yaml
    cart: [A.yaml, B.yaml]
  urls:
    /products: A.yaml
    /orders: B.yaml
  hosts:
    example.com: A.yaml
    another.com: B.yaml
4. Kullanım Senaryoları
4.1 Temel Kullanım

Web sayfasında kütüphane yüklenir
Kütüphane mevcut URL'yi analiz eder
Backend'den ilgili konfigürasyonları alır
DOM manipülasyonlarını sırasıyla uygular

4.2 A/B Testing

Farklı host'lar için farklı konfigürasyonlar
Kullanıcı segmentasyonuna göre içerik değişimi

4.3 Content Management

Reklam banner'larını kaldırma
İçerik güncelleme
Layout değişiklikleri

5. Performans Gereksinimleri
5.1 Frontend

Yükleme Süresi: <100ms
DOM Manipülasyon: <50ms
Memory Footprint: Minimal

5.2 Backend

Response Time: <200ms
Concurrent Users: En az 100
Availability: %99+

6. Güvenlik Gereksinimleri
6.1 Input Validation

XSS koruması
HTML sanitization
CSS injection koruması

6.2 API Security

Authentication (bonus)
Rate limiting
CORS konfigürasyonu

7. Test Gereksinimleri
7.1 Unit Tests

Her aksiyon tipi için test
Edge case'ler
Error handling

7.2 Integration Tests

Frontend-Backend entegrasyonu
End-to-end senaryolar

8. Bonus Özellikler
8.1 Gelişmiş Konfigürasyon

Conditional logic
Dynamic values
User context bazlı konfigürasyon

8.2 Monitoring

Performance metrics
Error logging
Usage analytics

9. Delivery Gereksinimleri
9.1 Kod Yapısı

Git repository
Clear commit history
Comprehensive README
Code documentation

9.2 Dokümantasyon

API documentation
Usage examples
Setup instructions
Architecture overview

10. Değerlendirme Kriterleri

Functionality (40%): Tüm aksiyonların çalışması
Backend Code Quality (10%): Kod yapısı ve maintainability
Frontend Code Quality (10%): JavaScript kod kalitesi
Scalability (20%): Gelecek geliştirmeler için uygunluk
Bonus Features (20%): Ekstra özellikler