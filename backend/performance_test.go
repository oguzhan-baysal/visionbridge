package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Backend Response Time Test - <200ms hedefi
func TestBackendResponseTime(t *testing.T) {
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

	tests := []struct {
		name           string
		method         string
		url            string
		body           interface{}
		expectedStatus int
		maxResponseTime time.Duration
	}{
		{
			name:           "GET All Configs",
			method:         "GET",
			url:            "/api/configuration/all",
			expectedStatus: 200,
			maxResponseTime: 200 * time.Millisecond,
		},
		{
			name:           "GET Single Config",
			method:         "GET", 
			url:            "/api/configuration/demo",
			expectedStatus: 200,
			maxResponseTime: 200 * time.Millisecond,
		},
		{
			name:           "POST New Config",
			method:         "POST",
			url:            "/api/configuration",
			body: map[string]interface{}{
				"id": "test-performance",
				"actions": []interface{}{
					map[string]interface{}{
						"type": "remove",
						"selector": ".test",
					},
				},
			},
			expectedStatus: 201,
			maxResponseTime: 200 * time.Millisecond,
		},
		{
			name:           "GET Specific Config",
			method:         "GET",
			url:            "/api/specific?host=localhost&url=/test",
			expectedStatus: 404, // Normal, çünkü test config yok
			maxResponseTime: 200 * time.Millisecond,
		},
	}

	fmt.Println("\n=== Backend Response Time Test Sonuçları ===")
	fmt.Printf("%-25s %-10s %-15s %-10s\n", "Test", "Method", "Response Time", "Status")
	fmt.Println(strings.Repeat("-", 70))

	allPassed := true
	totalTests := len(tests)
	passedTests := 0

	for _, test := range tests {
		// Request body hazırla
		var reqBody *bytes.Buffer
		if test.body != nil {
			jsonBody, _ := json.Marshal(test.body)
			reqBody = bytes.NewBuffer(jsonBody)
		} else {
			reqBody = bytes.NewBuffer(nil)
		}

		// Request oluştur
		req := httptest.NewRequest(test.method, test.url, reqBody)
		if test.body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		
		// Response recorder
		w := httptest.NewRecorder()

		// Zamanı ölç
		start := time.Now()
		handler.ServeHTTP(w, req)
		responseTime := time.Since(start)

		// Sonuçları kontrol et
		status := "✅ PASS"
		if responseTime > test.maxResponseTime {
			status = "❌ FAIL"
			allPassed = false
		} else {
			passedTests++
		}

		fmt.Printf("%-25s %-10s %-15s %-10s\n", 
			test.name, 
			test.method, 
			fmt.Sprintf("%v", responseTime),
			status)

		// Detaylı log
		t.Logf("%s: %s %s - Response Time: %v (Max: %v) - Status: %d", 
			status, test.method, test.url, responseTime, test.maxResponseTime, w.Code)
	}

	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Toplam: %d/%d test geçti\n", passedTests, totalTests)
	
	if allPassed {
		fmt.Println("🎉 Tüm response time testleri başarılı! (<200ms)")
	} else {
		fmt.Println("⚠️  Bazı testler 200ms hedefini aştı!")
	}

	// Test başarısızlığı durumunda fail et
	if !allPassed {
		t.Errorf("Response time testleri başarısız: %d/%d test geçti", passedTests, totalTests)
	}
}

// Benchmark testleri
func BenchmarkGetAllConfigs(b *testing.B) {
	router := mux.NewRouter()
	router.HandleFunc("/api/configuration/all", handleGetAllConfigs).Methods("GET")
	
	req := httptest.NewRequest("GET", "/api/configuration/all", nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetSingleConfig(b *testing.B) {
	router := mux.NewRouter()
	router.HandleFunc("/api/configuration/{id}", handleGetConfig).Methods("GET")
	
	req := httptest.NewRequest("GET", "/api/configuration/demo", nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkPostConfig(b *testing.B) {
	router := mux.NewRouter()
	router.HandleFunc("/api/configuration", handlePostConfig).Methods("POST")
	
	testConfig := map[string]interface{}{
		"id": "benchmark-test",
		"actions": []interface{}{
			map[string]interface{}{
				"type": "remove",
				"selector": ".benchmark",
			},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonBody, _ := json.Marshal(testConfig)
		req := httptest.NewRequest("POST", "/api/configuration", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// Load test simülasyonu
func TestLoadSimulation(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/configuration/all", handleGetAllConfigs).Methods("GET")
	
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := c.Handler(router)

	// 50 eşzamanlı request simülasyonu
	concurrentRequests := 50
	results := make(chan time.Duration, concurrentRequests)
	
	fmt.Printf("\n=== Load Test Simülasyonu (%d eşzamanlı request) ===\n", concurrentRequests)
	
	start := time.Now()
	
	for i := 0; i < concurrentRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/configuration/all", nil)
			w := httptest.NewRecorder()
			
			requestStart := time.Now()
			handler.ServeHTTP(w, req)
			requestTime := time.Since(requestStart)
			
			results <- requestTime
		}()
	}
	
	// Sonuçları topla
	var totalTime time.Duration
	var maxTime time.Duration
	var minTime time.Duration = time.Hour // Başlangıç değeri
	slowRequests := 0
	
	for i := 0; i < concurrentRequests; i++ {
		responseTime := <-results
		totalTime += responseTime
		
		if responseTime > maxTime {
			maxTime = responseTime
		}
		if responseTime < minTime {
			minTime = responseTime
		}
		if responseTime > 200*time.Millisecond {
			slowRequests++
		}
	}
	
	totalTestTime := time.Since(start)
	avgTime := totalTime / time.Duration(concurrentRequests)
	
	fmt.Printf("Toplam Test Süresi: %v\n", totalTestTime)
	fmt.Printf("Ortalama Response Time: %v\n", avgTime)
	fmt.Printf("En Hızlı Response: %v\n", minTime)
	fmt.Printf("En Yavaş Response: %v\n", maxTime)
	fmt.Printf("200ms'yi Aşan Request: %d/%d\n", slowRequests, concurrentRequests)
	
	if avgTime < 200*time.Millisecond {
		fmt.Println("✅ Ortalama response time hedefi karşılandı!")
	} else {
		fmt.Println("❌ Ortalama response time hedefi aşıldı!")
		t.Errorf("Ortalama response time %v > 200ms", avgTime)
	}
	
	if slowRequests == 0 {
		fmt.Println("🎉 Tüm requestler 200ms altında!")
	} else {
		fmt.Printf("⚠️  %d request 200ms'yi aştı\n", slowRequests)
	}
} 