package main

import (
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
	Base         string            `yaml:"base"`
	Environments []EnvironmentLink `yaml:"environments"`
	Shards       Shards            `yaml:"shards"`
	Customers    []CustomerGroup   `yaml:"customers"`
	Tabs         []Tab             `yaml:"tabs"`
}

type CustomerGroup struct {
	Shard   string   `yaml:"shard"`
	Tenants []string `yaml:"tenants"`
}

type EnvironmentLink struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type Shards struct {
	Designation []string    `yaml:"designation"`
	Items       []ShardItem `yaml:"items"`
}

type ShardItem struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	Icon string `yaml:"icon"`
}

type Tab struct {
	Name  string    `yaml:"name"`
	Items []TabItem `yaml:"items"`
}

type TabItem struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
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
}

type ExpandedCustomer struct {
	Name  string
	Links []ExpandedShardItem
}

type CustomerShardGroup struct {
	ShardName string
	Customers []ExpandedCustomer
}

func (c *Config) ExpandShards() []ShardGroup {
	var groups []ShardGroup

	for _, shard := range c.Shards.Designation {
		var items []ExpandedShardItem

		base := strings.TrimPrefix(c.Base, "pfn.")

		for _, item := range c.Shards.Items {
			url := fmt.Sprintf("https://%s.%s/%s", shard, base, item.Path)
			items = append(items, ExpandedShardItem{
				Name: item.Name,
				URL:  url,
				Icon: item.Icon,
			})
		}

		groups = append(groups, ShardGroup{
			ShardName: shard,
			Items:     items,
		})
	}

	return groups
}

func (c *Config) ExpandCustomers() []CustomerShardGroup {
	var groups []CustomerShardGroup

	base := strings.TrimPrefix(c.Base, "pfn.")

	for _, group := range c.Customers {
		var customers []ExpandedCustomer

		for _, tenant := range group.Tenants {
			var links []ExpandedShardItem

			for _, item := range c.Shards.Items {
				url := fmt.Sprintf(
					"https://%s.%s/%s",
					tenant,
					base,
					item.Path,
				)

				links = append(links, ExpandedShardItem{
					Name: item.Name,
					URL:  url,
					Icon: item.Icon,
				})
			}

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

	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatal("Error parsing template:", err)
	}

	for i := range cfg.Tabs {
		for j := range cfg.Tabs[i].Items {
			cfg.Tabs[i].Items[j].URL = buildURL(cfg.Base, cfg.Tabs[i].Items[j].URL)
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

		err := tmpl.ExecuteTemplate(w, "index.html", data)
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
