package main

import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"path/filepath"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"regexp"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"time"
)

// Config yapısı (örnek, dinamik alanlar için map[string]interface{})
type Config map[string]interface{}

// Pages bazlı konfigürasyon yapısı
type PagesConfig struct {
	ID         string                 `yaml:"id" json:"id"`
	Name       string                 `yaml:"name" json:"name"`
	Datasource PagesDataSource        `yaml:"datasource" json:"datasource"`
	Actions    []interface{}          `yaml:"actions" json:"actions"`
	Metadata   map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type PagesDataSource struct {
	Pages map[string]interface{} `yaml:"pages,omitempty" json:"pages,omitempty"`
	URLs  map[string]interface{} `yaml:"urls,omitempty" json:"urls,omitempty"`
	Hosts map[string]interface{} `yaml:"hosts,omitempty" json:"hosts,omitempty"`
}

// Yardımcı: configs klasörü yolu
const configDir = "configs"

// Yardımcı: id'den dosya yolu
func configPath(id string) string {
	return filepath.Join(configDir, id+".yaml")
}

// Yardımcı: spesifik config dosya yolu
func specificConfigPath(id string) string {
	return filepath.Join(configDir, "specific_"+id+".yaml")
}

// Yardımcı: pages config dosya yolu
func pagesConfigPath(id string) string {
	return filepath.Join(configDir, "pages_"+id+".yaml")
}

// Pages konfigürasyonunu yükle
func loadPagesConfig(id string) (*PagesConfig, error) {
	b, err := ioutil.ReadFile(pagesConfigPath(id))
	if err != nil {
		return nil, err
	}
	var cfg PagesConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Pages konfigürasyonunu kaydet
func savePagesConfig(cfg *PagesConfig) error {
	// Actions validasyonu
	if cfg.Actions != nil {
		valid, sanitized := validateAndSanitizeActions(cfg.Actions)
		if !valid {
			return fmt.Errorf("geçersiz actions")
		}
		cfg.Actions = sanitized
	}
	
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(pagesConfigPath(cfg.ID), b, 0644)
}

// Tüm pages konfigürasyonlarını listele
func getAllPagesConfigs() ([]PagesConfig, error) {
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		return nil, err
	}
	
	var configs []PagesConfig
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" && 
		   len(file.Name()) > 6 && file.Name()[:6] == "pages_" {
			id := file.Name()[6 : len(file.Name())-5] // "pages_" ve ".yaml" kaldır
			if cfg, err := loadPagesConfig(id); err == nil {
				configs = append(configs, *cfg)
			}
		}
	}
	return configs, nil
}

// Basit HTML sanitizer: <script>, <style> ve on* eventlerini kaldırır
func sanitizeHTML(input string) string {
	reScript := regexp.MustCompile(`(?i)<script.*?>.*?</script>`) // script tag
	reStyle := regexp.MustCompile(`(?i)<style.*?>.*?</style>`)   // style tag
	reOnEventDouble := regexp.MustCompile(`(?i)on\w+\s*=\s*"[^"]*"`)
	reOnEventSingle := regexp.MustCompile(`(?i)on\w+\s*=\s*'[^']*'`)
	out := reScript.ReplaceAllString(input, "")
	out = reStyle.ReplaceAllString(out, "")
	out = reOnEventDouble.ReplaceAllString(out, "")
	out = reOnEventSingle.ReplaceAllString(out, "")
	return out
}

// Basit CSS selector validasyonu (class, id, tag)
func isValidSelector(sel string) bool {
	if sel == "" { return false }
	// Sadece harf, rakam, . # ve - _ : [ ] karakterlerine izin ver
	re := regexp.MustCompile(`^[a-zA-Z0-9\.\#\-\_\:\[\]=\'\"]+$`)
	return re.MatchString(sel)
}

// Actions validasyonu ve sanitizasyonu
func validateAndSanitizeActions(actions []interface{}) (bool, []interface{}) {
	var sanitized []interface{}
	for _, a := range actions {
		act, ok := a.(map[string]interface{})
		if !ok { return false, nil }
		typeStr, _ := act["type"].(string)
		switch typeStr {
		case "remove", "replace", "insert":
			// selector veya target kontrolü
			if sel, ok := act["selector"].(string); ok && !isValidSelector(sel) {
				return false, nil
			}
			if tgt, ok := act["target"].(string); ok && !isValidSelector(tgt) {
				return false, nil
			}
			// HTML içeriği sanitize et
			if html, ok := act["newElement"].(string); ok {
				act["newElement"] = sanitizeHTML(html)
			}
			if html, ok := act["element"].(string); ok {
				act["element"] = sanitizeHTML(html)
			}
		case "alter":
			// metin değişimi, ek kontrol gerekmez
		default:
			return false, nil
		}
		sanitized = append(sanitized, act)
	}
	return true, sanitized
}

// GET /api/configuration/all
func handleGetAllConfigs(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Config klasörü okunamadı"}`))
		return
	}
	var configs []Config
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" {
			b, err := ioutil.ReadFile(filepath.Join(configDir, file.Name()))
			if err != nil { continue }
			var cfg Config
			if err := yaml.Unmarshal(b, &cfg); err != nil { continue }
			configs = append(configs, cfg)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// GET /api/configuration/{id}
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	b, err := ioutil.ReadFile(configPath(id))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Config bulunamadı"}`))
		return
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "YAML parse hatası"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// POST /api/configuration
func handlePostConfig(w http.ResponseWriter, r *http.Request) {
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "JSON parse hatası"}`))
		return
	}
	id, ok := cfg["id"].(string)
	if !ok || id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "id alanı zorunlu"}`))
		return
	}
	// actions validasyonu
	actions, ok := cfg["actions"].([]interface{})
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions array zorunlu"}`))
		return
	}
	valid, sanitized := validateAndSanitizeActions(actions)
	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions geçersiz veya tehlikeli içerik"}`))
		return
	}
	cfg["actions"] = sanitized
	b, err := yaml.Marshal(cfg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "YAML'e çevirilemedi"}`))
		return
	}
	if err := ioutil.WriteFile(configPath(id), b, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Dosya yazılamadı"}`))
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Config eklendi"}`))
}

// PUT /api/configuration/{id}
func handlePutConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "JSON parse hatası"}`))
		return
	}
	// actions validasyonu
	actions, ok := cfg["actions"].([]interface{})
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions array zorunlu"}`))
		return
	}
	valid, sanitized := validateAndSanitizeActions(actions)
	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions geçersiz veya tehlikeli içerik"}`))
		return
	}
	cfg["actions"] = sanitized
	b, err := yaml.Marshal(cfg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "YAML'e çevirilemedi"}`))
		return
	}
	if err := ioutil.WriteFile(configPath(id), b, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Dosya yazılamadı"}`))
		return
	}
	w.Write([]byte(`{"message": "Config güncellendi"}`))
}

// DELETE /api/configuration/{id}
func handleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := os.Remove(configPath(id)); err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Config silinemedi"}`))
		return
	}
	w.Write([]byte(`{"message": "Config silindi"}`))
}

// GET /api/specific?id=...&host=...&url=...
func handleGetSpecificConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	host := r.URL.Query().Get("host")
	url := r.URL.Query().Get("url")

	// 1. id ile arama
	if id != "" {
		b, err := ioutil.ReadFile(specificConfigPath(id))
		if err == nil {
			var cfg Config
			if err := yaml.Unmarshal(b, &cfg); err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(cfg)
				return
			}
		}
	}

	// 2. host/url ile arama (tüm spesifikleri tara)
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Config klasörü okunamadı"}`))
		return
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" && filepath.HasPrefix(file.Name(), "specific_") {
			b, err := ioutil.ReadFile(filepath.Join(configDir, file.Name()))
			if err != nil { continue }
			var cfg Config
			if err := yaml.Unmarshal(b, &cfg); err != nil { continue }
			if ds, ok := cfg["datasource"].(map[string]interface{}); ok {
				if hosts, ok := ds["hosts"].(map[string]interface{}); ok {
					if _, ok := hosts[host]; ok {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(cfg)
						return
					}
				}
				if urls, ok := ds["urls"].(map[string]interface{}); ok {
					if _, ok := urls[url]; ok {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(cfg)
						return
					}
				}
			}
		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "Spesifik config bulunamadı"}`))
}

// GET /api/specific/{id}
func handleGetSpecificById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	b, err := ioutil.ReadFile(specificConfigPath(id))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Spesifik config bulunamadı"}`))
		return
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "YAML parse hatası"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// POST /api/specific
func handlePostSpecific(w http.ResponseWriter, r *http.Request) {
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "JSON parse hatası"}`))
		return
	}
	id, ok := cfg["id"].(string)
	if !ok || id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "id alanı zorunlu"}`))
		return
	}
	// actions validasyonu
	actions, ok := cfg["actions"].([]interface{})
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions array zorunlu"}`))
		return
	}
	valid, sanitized := validateAndSanitizeActions(actions)
	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions geçersiz veya tehlikeli içerik"}`))
		return
	}
	cfg["actions"] = sanitized
	b, err := yaml.Marshal(cfg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "YAML'e çevirilemedi"}`))
		return
	}
	if err := ioutil.WriteFile(specificConfigPath(id), b, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Dosya yazılamadı"}`))
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Spesifik config eklendi"}`))
}

// PUT /api/specific/{id}
func handlePutSpecific(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "JSON parse hatası"}`))
		return
	}
	// actions validasyonu
	actions, ok := cfg["actions"].([]interface{})
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions array zorunlu"}`))
		return
	}
	valid, sanitized := validateAndSanitizeActions(actions)
	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "actions geçersiz veya tehlikeli içerik"}`))
		return
	}
	cfg["actions"] = sanitized
	b, err := yaml.Marshal(cfg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "YAML'e çevirilemedi"}`))
		return
	}
	if err := ioutil.WriteFile(specificConfigPath(id), b, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Dosya yazılamadı"}`))
		return
	}
	w.Write([]byte(`{"message": "Spesifik config güncellendi"}`))
}

// DELETE /api/specific/{id}
func handleDeleteSpecific(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := os.Remove(specificConfigPath(id)); err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Spesifik config silinemedi"}`))
		return
	}
	w.Write([]byte(`{"message": "Spesifik config silindi"}`))
}

// Pages Configuration Endpoints

// GET /api/pages/all
func handleGetAllPagesConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := getAllPagesConfigs()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Pages config klasörü okunamadı"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// GET /api/pages/{id}
func handleGetPagesConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	cfg, err := loadPagesConfig(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Pages config bulunamadı"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// POST /api/pages
func handlePostPagesConfig(w http.ResponseWriter, r *http.Request) {
	var cfg PagesConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "JSON parse hatası"}`))
		return
	}
	
	if cfg.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "id alanı zorunlu"}`))
		return
	}
	
	if err := savePagesConfig(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "Pages config eklendi"}`))
}

// PUT /api/pages/{id}
func handlePutPagesConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var cfg PagesConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "JSON parse hatası"}`))
		return
	}
	
	cfg.ID = id // URL'den gelen ID'yi kullan
	
	if err := savePagesConfig(&cfg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
		return
	}
	
	w.Write([]byte(`{"message": "Pages config güncellendi"}`))
}

// DELETE /api/pages/{id}
func handleDeletePagesConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := os.Remove(pagesConfigPath(id)); err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Pages config silinemedi"}`))
		return
	}
	w.Write([]byte(`{"message": "Pages config silindi"}`))
}

// GET /api/pages/resolve - Query parametrelerine göre uygun pages config'i döndür
func handleResolvePagesConfig(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	url := r.URL.Query().Get("url")
	page := r.URL.Query().Get("page")
	
	if host == "" && url == "" && page == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "host, url veya page parametresi gerekli"}`))
		return
	}
	
	configs, err := getAllPagesConfigs()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Pages config okunamadı"}`))
		return
	}
	
	// Öncelik sırası: page > url > host
	for _, cfg := range configs {
		// Page bazlı eşleşme
		if page != "" && cfg.Datasource.Pages != nil {
			if configRef, exists := cfg.Datasource.Pages[page]; exists {
				result := map[string]interface{}{
					"config": cfg,
					"matched_by": "page",
					"matched_value": page,
					"config_ref": configRef,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
				return
			}
		}
		
		// URL bazlı eşleşme
		if url != "" && cfg.Datasource.URLs != nil {
			if configRef, exists := cfg.Datasource.URLs[url]; exists {
				result := map[string]interface{}{
					"config": cfg,
					"matched_by": "url",
					"matched_value": url,
					"config_ref": configRef,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
				return
			}
		}
		
		// Host bazlı eşleşme
		if host != "" && cfg.Datasource.Hosts != nil {
			if configRef, exists := cfg.Datasource.Hosts[host]; exists {
				result := map[string]interface{}{
					"config": cfg,
					"matched_by": "host",
					"matched_value": host,
					"config_ref": configRef,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
				return
			}
		}
	}
	
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "Eşleşen pages config bulunamadı"}`))
}

// Log middleware
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		fmt.Printf("[%s] %s %s %d %s\n", start.Format(time.RFC3339), r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	router := mux.NewRouter()

	// Test endpoint
	router.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "pong"}`))
	})

	router.HandleFunc("/api/configuration/all", handleGetAllConfigs).Methods("GET")
	router.HandleFunc("/api/configuration/{id}", handleGetConfig).Methods("GET")
	router.HandleFunc("/api/configuration", handlePostConfig).Methods("POST")
	router.HandleFunc("/api/configuration/{id}", handlePutConfig).Methods("PUT")
	router.HandleFunc("/api/configuration/{id}", handleDeleteConfig).Methods("DELETE")

	router.HandleFunc("/api/specific", handleGetSpecificConfig).Methods("GET")
	router.HandleFunc("/api/specific/{id}", handleGetSpecificById).Methods("GET")
	router.HandleFunc("/api/specific", handlePostSpecific).Methods("POST")
	router.HandleFunc("/api/specific/{id}", handlePutSpecific).Methods("PUT")
	router.HandleFunc("/api/specific/{id}", handleDeleteSpecific).Methods("DELETE")

	// Pages Configuration Routes
	router.HandleFunc("/api/pages/all", handleGetAllPagesConfigs).Methods("GET")
	router.HandleFunc("/api/pages/resolve", handleResolvePagesConfig).Methods("GET")
	router.HandleFunc("/api/pages/{id}", handleGetPagesConfig).Methods("GET")
	router.HandleFunc("/api/pages", handlePostPagesConfig).Methods("POST")
	router.HandleFunc("/api/pages/{id}", handlePutPagesConfig).Methods("PUT")
	router.HandleFunc("/api/pages/{id}", handleDeletePagesConfig).Methods("DELETE")

	// CORS ayarları
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Log middleware'i ekle
	handler := logMiddleware(c.Handler(router))

	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
} 