package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Item struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Icon string `yaml:"icon"`
}

type Tab struct {
	Name  string `yaml:"name"`
	Items []Item `yaml:"items"`
}

type Config struct {
	Title       string `yaml:"title"`
	Environment string `yaml:"environment"`
	Base        string `yaml:"base"`
	Tabs        []Tab  `yaml:"tabs"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	cfg, err := loadConfig("cfg/config.yaml")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatal("Error parsing template:", err)
	}

	for i := range cfg.Tabs {
		for j := range cfg.Tabs[i].Items {
			cfg.Tabs[i].Items[j].URL = buildURL(cfg.Base, cfg.Tabs[i].Items[j].URL)
		}
	}

	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("resources"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.ExecuteTemplate(w, "index.html", cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	log.Println("Server running at port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func buildURL(base, u string) string {
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u // absolute URLs used as-is
	}
	return "https://" + u + "." + base // relative URL expanded
}
