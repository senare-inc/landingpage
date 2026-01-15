package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Config struct {
	Title        string            `yaml:"title"`
	Environment  string            `yaml:"environment"`
	FQDN         string            `yaml:"fqdn"`
	Environments []EnvironmentLink `yaml:"environments"`
	EnvColor     string            // computed at runtime
	Shards       Shards            `yaml:"shards"`
	Customers    []CustomerGroup   `yaml:"customers"`
	Tabs         []Tab             `yaml:"tabs"`
}

type CustomerGroup struct {
	Shard   string   `yaml:"shard"`
	Tenants []string `yaml:"tenants"`
}

type EnvironmentLink struct {
	Name  string `yaml:"name"`
	URL   string `yaml:"url"`
	Color string `yaml:"color"`
}

type Shards struct {
	Items []Item `yaml:"items"`
}

type Tab struct {
	Index int
	Name  string `yaml:"name"`
	Items []Item `yaml:"items"`
}

type Item struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`  // used by tabs
	Path string `yaml:"path,omitempty"` // used by shards
	Icon string `yaml:"icon"`
}

type ExpandedShardItem struct {
	Name string
	URL  string
	Icon string
}

type ShardGroup struct {
	ShardName string
	Items     []ExpandedShardItem
	DataShard string
}

type ExpandedCustomer struct {
	Name  string
	Links []ExpandedShardItem
}

type CustomerShardGroup struct {
	ShardName string
	Customers []ExpandedCustomer
}

// ======================================================
// Infer shard list from customers (replaces designation)
// ======================================================

func (c *Config) shardListFromCustomers() []string {
	seen := map[string]bool{}
	var result []string

	for _, g := range c.Customers {
		if !seen[g.Shard] {
			seen[g.Shard] = true
			result = append(result, g.Shard)
		}
	}

	return result
}

//======================================================
// Infer base from FQDN i.e. drop pfn
// ======================================================

func (c *Config) base() string {
	return strings.TrimPrefix(c.FQDN, "pfn.")
}

// ======================================================

func (c *Config) ExpandShards() []ShardGroup {
	var groups []ShardGroup

	for _, shard := range c.shardListFromCustomers() {
		var items []ExpandedShardItem

		for _, item := range c.Shards.Items {
			url := fmt.Sprintf("https://%s.%s/%s", shard, c.base(), item.Path)

			items = append(items, ExpandedShardItem{
				Name: item.Name,
				URL:  url,
				Icon: item.Icon,
			})
		}

		groups = append(groups, ShardGroup{
			ShardName: shard,
			Items:     items,
			DataShard: strings.ToLower(strings.ReplaceAll(shard, " ", "-")),
		})
	}

	return groups
}

func (c *Config) ExpandCustomers() []CustomerShardGroup {
	var groups []CustomerShardGroup

	for _, group := range c.Customers {
		var customers []ExpandedCustomer

		for _, tenant := range group.Tenants {
			var links []ExpandedShardItem

			url := fmt.Sprintf(
				"https://%s.%s/%s",
				tenant,
				c.base(),
				"wanda",
			)

			links = append(links, ExpandedShardItem{
				Name: tenant,
				URL:  url,
				Icon: "fish.svg",
			})

			customers = append(customers, ExpandedCustomer{
				Name:  tenant,
				Links: links,
			})
		}

		groups = append(groups, CustomerShardGroup{
			ShardName: group.Shard,
			Customers: customers,
		})
	}

	return groups
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

	// Resolve color for current environment
	for _, e := range cfg.Environments {
		if e.Name == cfg.Environment {
			cfg.EnvColor = e.Color
		}
	}

	// fallback
	if cfg.EnvColor == "" {
		cfg.EnvColor = "#1e40af"
	}

	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatal("Error parsing template:", err)
	}

	for i := range cfg.Tabs {
		cfg.Tabs[i].Index = i
		for j := range cfg.Tabs[i].Items {
			cfg.Tabs[i].Items[j].URL = buildURL(cfg.FQDN, cfg.Tabs[i].Items[j].URL)
		}
	}

	expandedShards := cfg.ExpandShards()
	expandedCustomers := cfg.ExpandCustomers()

	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("resources"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			*Config
			ExpandedShards    []ShardGroup
			ExpandedCustomers []CustomerShardGroup
		}{
			Config:            cfg,
			ExpandedShards:    expandedShards,
			ExpandedCustomers: expandedCustomers,
		}

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "index.html", data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Println("Template execution error:", err)
			return
		}
		w.Write(buf.Bytes())
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
