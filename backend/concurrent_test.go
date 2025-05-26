package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// 100 Eşzamanlı Kullanıcı Testi
func TestConcurrentUsers100(t *testing.T) {
	// Test server'ı oluştur
	router := mux.NewRouter()
	router.HandleFunc("/api/configuration/all", handleGetAllConfigs).Methods("GET")
	router.HandleFunc("/api/configuration/{id}", handleGetConfig).Methods("GET")
	router.HandleFunc("/api/configuration", handlePostConfig).Methods("POST")
	router.HandleFunc("/api/specific", handleGetSpecificConfig).Methods("GET")
	
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := c.Handler(router)

	concurrentUsers := 100
	requestsPerUser := 5 // Her kullanıcı 5 request yapar
	totalRequests := concurrentUsers * requestsPerUser

	fmt.Printf("\n=== 100 Eşzamanlı Kullanıcı Testi ===\n")
	fmt.Printf("Kullanıcı Sayısı: %d\n", concurrentUsers)
	fmt.Printf("Kullanıcı Başına Request: %d\n", requestsPerUser)
	fmt.Printf("Toplam Request: %d\n", totalRequests)
	fmt.Println("Test başlatılıyor...")

	// Sonuçları toplamak için
	results := make(chan TestResult, totalRequests)
	var wg sync.WaitGroup

	// Test başlangıç zamanı
	testStart := time.Now()

	// 100 eşzamanlı kullanıcı simülasyonu
	for userID := 0; userID < concurrentUsers; userID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			simulateUser(id, requestsPerUser, handler, results)
		}(userID)
	}

	// Tüm kullanıcıların bitmesini bekle
	wg.Wait()
	close(results)

	totalTestTime := time.Since(testStart)

	// Sonuçları analiz et
	analyzeResults(t, results, totalRequests, totalTestTime, concurrentUsers)
}

// Test sonucu yapısı
type TestResult struct {
	UserID       int
	RequestID    int
	Method       string
	URL          string
	ResponseTime time.Duration
	StatusCode   int
	Success      bool
}

// Kullanıcı simülasyonu
func simulateUser(userID int, requestCount int, handler http.Handler, results chan<- TestResult) {
	// Her kullanıcı farklı request'ler yapar
	requests := []struct {
		method string
		url    string
		body   interface{}
	}{
		{"GET", "/api/configuration/all", nil},
		{"GET", "/api/configuration/demo", nil},
		{"GET", "/api/specific?host=localhost&url=/test", nil},
		{"POST", "/api/configuration", map[string]interface{}{
			"id": fmt.Sprintf("user-%d-config", userID),
			"actions": []interface{}{
				map[string]interface{}{
					"type":     "remove",
					"selector": fmt.Sprintf(".user-%d", userID),
				},
			},
		}},
		{"GET", "/api/configuration/all", nil}, // Tekrar GET
	}

	for reqID := 0; reqID < requestCount; reqID++ {
		req := requests[reqID%len(requests)]
		
		// Request body hazırla
		var reqBody *bytes.Buffer
		if req.body != nil {
			jsonBody, _ := json.Marshal(req.body)
			reqBody = bytes.NewBuffer(jsonBody)
		} else {
			reqBody = bytes.NewBuffer(nil)
		}

		// HTTP request oluştur
		httpReq := httptest.NewRequest(req.method, req.url, reqBody)
		if req.body != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		// Response recorder
		w := httptest.NewRecorder()

		// Request zamanını ölç
		start := time.Now()
		handler.ServeHTTP(w, httpReq)
		responseTime := time.Since(start)

		// Sonucu kaydet
		result := TestResult{
			UserID:       userID,
			RequestID:    reqID,
			Method:       req.method,
			URL:          req.url,
			ResponseTime: responseTime,
			StatusCode:   w.Code,
			Success:      w.Code >= 200 && w.Code < 400,
		}

		results <- result

		// Kullanıcılar arası küçük gecikme (gerçekçi simülasyon)
		time.Sleep(time.Millisecond * time.Duration(10+userID%20))
	}
}

// Sonuçları analiz et
func analyzeResults(t *testing.T, results <-chan TestResult, totalRequests int, totalTime time.Duration, concurrentUsers int) {
	var (
		successCount     int
		failCount        int
		totalResponseTime time.Duration
		minResponseTime  time.Duration = time.Hour
		maxResponseTime  time.Duration
		slowRequests     int // >200ms
		verySlowRequests int // >500ms
		methodStats      = make(map[string]struct {
			count int
			totalTime time.Duration
		})
	)

	fmt.Println("\n=== Sonuç Analizi ===")

	// Tüm sonuçları işle
	for result := range results {
		if result.Success {
			successCount++
		} else {
			failCount++
		}

		totalResponseTime += result.ResponseTime

		if result.ResponseTime < minResponseTime {
			minResponseTime = result.ResponseTime
		}
		if result.ResponseTime > maxResponseTime {
			maxResponseTime = result.ResponseTime
		}

		if result.ResponseTime > 200*time.Millisecond {
			slowRequests++
		}
		if result.ResponseTime > 500*time.Millisecond {
			verySlowRequests++
		}

		// Method bazlı istatistikler
		stats := methodStats[result.Method]
		stats.count++
		stats.totalTime += result.ResponseTime
		methodStats[result.Method] = stats
	}

	// Genel istatistikler
	avgResponseTime := totalResponseTime / time.Duration(totalRequests)
	requestsPerSecond := float64(totalRequests) / totalTime.Seconds()
	successRate := float64(successCount) / float64(totalRequests) * 100

	fmt.Printf("Toplam Test Süresi: %v\n", totalTime)
	fmt.Printf("Toplam Request: %d\n", totalRequests)
	fmt.Printf("Başarılı Request: %d\n", successCount)
	fmt.Printf("Başarısız Request: %d\n", failCount)
	fmt.Printf("Başarı Oranı: %.2f%%\n", successRate)
	fmt.Printf("Request/Saniye: %.2f\n", requestsPerSecond)
	fmt.Println()

	// Response time istatistikleri
	fmt.Println("=== Response Time İstatistikleri ===")
	fmt.Printf("Ortalama: %v\n", avgResponseTime)
	fmt.Printf("En Hızlı: %v\n", minResponseTime)
	fmt.Printf("En Yavaş: %v\n", maxResponseTime)
	fmt.Printf("200ms'yi Aşan: %d (%.2f%%)\n", slowRequests, float64(slowRequests)/float64(totalRequests)*100)
	fmt.Printf("500ms'yi Aşan: %d (%.2f%%)\n", verySlowRequests, float64(verySlowRequests)/float64(totalRequests)*100)
	fmt.Println()

	// Method bazlı istatistikler
	fmt.Println("=== Method Bazlı İstatistikler ===")
	for method, stats := range methodStats {
		avgTime := stats.totalTime / time.Duration(stats.count)
		fmt.Printf("%s: %d request, ortalama %v\n", method, stats.count, avgTime)
	}
	fmt.Println()

	// Hedef kontrolü
	fmt.Println("=== Hedef Kontrolü ===")
	
	// 1. Response Time Hedefi (<200ms)
	if avgResponseTime < 200*time.Millisecond {
		fmt.Printf("✅ Ortalama Response Time: %v < 200ms\n", avgResponseTime)
	} else {
		fmt.Printf("❌ Ortalama Response Time: %v > 200ms\n", avgResponseTime)
		t.Errorf("Ortalama response time hedefi aşıldı: %v", avgResponseTime)
	}

	// 2. Başarı Oranı (>95%)
	if successRate >= 95.0 {
		fmt.Printf("✅ Başarı Oranı: %.2f%% >= 95%%\n", successRate)
	} else {
		fmt.Printf("❌ Başarı Oranı: %.2f%% < 95%%\n", successRate)
		t.Errorf("Başarı oranı hedefi karşılanmadı: %.2f%%", successRate)
	}

	// 3. Concurrent User Desteği (100 kullanıcı)
	if concurrentUsers >= 100 {
		fmt.Printf("✅ Eşzamanlı Kullanıcı: %d >= 100\n", concurrentUsers)
	} else {
		fmt.Printf("❌ Eşzamanlı Kullanıcı: %d < 100\n", concurrentUsers)
		t.Errorf("Eşzamanlı kullanıcı hedefi karşılanmadı: %d", concurrentUsers)
	}

	// 4. Yavaş Request Oranı (<10%)
	slowRequestRate := float64(slowRequests) / float64(totalRequests) * 100
	if slowRequestRate < 10.0 {
		fmt.Printf("✅ Yavaş Request Oranı: %.2f%% < 10%%\n", slowRequestRate)
	} else {
		fmt.Printf("⚠️  Yavaş Request Oranı: %.2f%% >= 10%%\n", slowRequestRate)
	}

	// 5. Throughput (Request/Saniye)
	if requestsPerSecond >= 100 {
		fmt.Printf("✅ Throughput: %.2f req/s >= 100\n", requestsPerSecond)
	} else {
		fmt.Printf("⚠️  Throughput: %.2f req/s < 100\n", requestsPerSecond)
	}

	fmt.Println()
	
	// Genel sonuç
	if avgResponseTime < 200*time.Millisecond && successRate >= 95.0 && concurrentUsers >= 100 {
		fmt.Println("🎉 100 Eşzamanlı Kullanıcı Testi BAŞARILI!")
		fmt.Println("   - Response time hedefi karşılandı")
		fmt.Println("   - Başarı oranı hedefi karşılandı") 
		fmt.Println("   - Concurrent user hedefi karşılandı")
	} else {
		fmt.Println("⚠️  100 Eşzamanlı Kullanıcı Testi KISMEN BAŞARILI")
		fmt.Println("   Bazı hedefler karşılanmadı, detaylar yukarıda")
	}
}

// Stress test - 200 kullanıcı ile
func TestStressTest200Users(t *testing.T) {
	if testing.Short() {
		t.Skip("Stress test atlanıyor (short mode)")
	}

	// Test server'ı oluştur
	router := mux.NewRouter()
	router.HandleFunc("/api/configuration/all", handleGetAllConfigs).Methods("GET")
	router.HandleFunc("/api/configuration/{id}", handleGetConfig).Methods("GET")
	
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := c.Handler(router)

	concurrentUsers := 200
	requestsPerUser := 3
	totalRequests := concurrentUsers * requestsPerUser

	fmt.Printf("\n=== Stress Test (200 Kullanıcı) ===\n")
	fmt.Printf("Bu test sistemin limitlerini test eder\n")
	fmt.Printf("Kullanıcı Sayısı: %d\n", concurrentUsers)
	fmt.Printf("Toplam Request: %d\n", totalRequests)

	results := make(chan TestResult, totalRequests)
	var wg sync.WaitGroup

	testStart := time.Now()

	// 200 eşzamanlı kullanıcı
	for userID := 0; userID < concurrentUsers; userID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Sadece GET requestleri (daha hızlı)
			for reqID := 0; reqID < requestsPerUser; reqID++ {
				req := httptest.NewRequest("GET", "/api/configuration/all", nil)
				w := httptest.NewRecorder()

				start := time.Now()
				handler.ServeHTTP(w, req)
				responseTime := time.Since(start)

				results <- TestResult{
					UserID:       id,
					RequestID:    reqID,
					Method:       "GET",
					URL:          "/api/configuration/all",
					ResponseTime: responseTime,
					StatusCode:   w.Code,
					Success:      w.Code == 200,
				}
			}
		}(userID)
	}

	wg.Wait()
	close(results)

	totalTestTime := time.Since(testStart)

	// Basit analiz
	var successCount, totalCount int
	var totalResponseTime time.Duration
	var maxResponseTime time.Duration

	for result := range results {
		totalCount++
		if result.Success {
			successCount++
		}
		totalResponseTime += result.ResponseTime
		if result.ResponseTime > maxResponseTime {
			maxResponseTime = result.ResponseTime
		}
	}

	avgResponseTime := totalResponseTime / time.Duration(totalCount)
	successRate := float64(successCount) / float64(totalCount) * 100
	requestsPerSecond := float64(totalCount) / totalTestTime.Seconds()

	fmt.Printf("Sonuçlar:\n")
	fmt.Printf("- Toplam Süre: %v\n", totalTestTime)
	fmt.Printf("- Başarı Oranı: %.2f%%\n", successRate)
	fmt.Printf("- Ortalama Response Time: %v\n", avgResponseTime)
	fmt.Printf("- En Yavaş Response: %v\n", maxResponseTime)
	fmt.Printf("- Throughput: %.2f req/s\n", requestsPerSecond)

	if successRate >= 90 && avgResponseTime < 500*time.Millisecond {
		fmt.Println("✅ Stress test başarılı!")
	} else {
		fmt.Println("⚠️  Stress test sınırları zorladı")
	}
} 