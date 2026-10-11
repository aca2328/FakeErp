package main

// Mock ERP modules: an HTML page and a JSON API per module, with fake data.
// Distinct paths make good targets for content switching, rewrites and
// per-path policies.

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
)

type module struct {
	Path, Title string
	Columns     []string
	row         func(r *rand.Rand, i int) []any
}

var (
	customers  = []string{"Acme Corp", "Globex", "Initech", "Umbrella", "Stark Industries", "Wayne Enterprises", "Soylent", "Hooli"}
	products   = []string{"Serveur 1U", "Switch 48p", "Licence ERP", "Ordinateur portable", "Écran 27\"", "Câble SFP+", "Onduleur", "Baie 42U"}
	warehouses = []string{"Paris", "Lyon", "Berlin", "Munich", "New York", "Austin"}
	accounts   = []string{"401 Fournisseurs", "411 Clients", "512 Banque", "606 Achats", "706 Ventes", "641 Salaires"}
	statuses   = []string{"Nouvelle", "En cours", "Expédiée", "Livrée", "Annulée"}
)

func pick(r *rand.Rand, s []string) string { return s[r.Intn(len(s))] }

var modules = []module{
	{"sales", "Gestion des ventes", []string{"Commande", "Client", "Montant (€)", "Statut"},
		func(r *rand.Rand, i int) []any {
			return []any{fmt.Sprintf("CMD-%05d", 10000+i), pick(r, customers), r.Intn(50000) + 100, pick(r, statuses)}
		}},
	{"stock", "Gestion des stocks", []string{"Référence", "Produit", "Quantité", "Entrepôt"},
		func(r *rand.Rand, i int) []any {
			return []any{fmt.Sprintf("SKU-%04d", 1000+i), pick(r, products), r.Intn(500), pick(r, warehouses)}
		}},
	{"accounting", "Comptabilité", []string{"Écriture", "Compte", "Débit (€)", "Crédit (€)"},
		func(r *rand.Rand, i int) []any {
			v := r.Intn(20000) + 10
			if r.Intn(2) == 0 {
				return []any{fmt.Sprintf("ECR-%05d", 50000+i), pick(r, accounts), v, 0}
			}
			return []any{fmt.Sprintf("ECR-%05d", 50000+i), pick(r, accounts), 0, v}
		}},
}

// rows returns n fake rows; a fixed seed keeps the data identical on every
// backend so only the "served by" field differs.
func (m module) rows(n int) [][]any {
	r := rand.New(rand.NewSource(int64(len(m.Path))))
	out := make([][]any, n)
	for i := range out {
		out[i] = m.row(r, i)
	}
	return out
}

var moduleTmpl = template.Must(template.New("module").Parse(`<!DOCTYPE html>
<html lang="fr"><head><meta charset="utf-8"><title>PRIMO SAS - {{.M.Title}}</title>
<style>
body { font-family: Arial, sans-serif; background-color: #f2f2f2; margin: 0; }
.header { background-color: {{.S.Color}}; color: #fff; padding: 20px; text-align: center; }
.container { max-width: 800px; margin: 20px auto; padding: 20px; background: #fff; }
table { border-collapse: collapse; width: 100%; }
th, td { border-bottom: 1px solid #ddd; padding: 6px; text-align: left; }
</style></head>
<body>
<div class="header"><h1>PRIMO SAS - {{.S.Name}} Server</h1></div>
<div class="container">
<h2>{{.M.Title}}</h2>
<p>Servi par {{.Host}} · <a href="/api/{{.M.Path}}">API JSON</a> · <a href="/">Accueil</a></p>
<table><tr>{{range .M.Columns}}<th>{{.}}</th>{{end}}</tr>
{{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>{{end}}
</table>
</div></body></html>`))

func moduleHandler(m module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		moduleTmpl.Execute(w, map[string]any{"M": m, "S": siteInfo, "Host": hostname, "Rows": m.rows(20)})
	}
}

// moduleAPIHandler returns ?limit=N rows (default 50, max 1000) as JSON.
func moduleAPIHandler(m module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows := m.rows(queryInt(r, "limit", 50, 1, 1000))
		items := make([]map[string]any, len(rows))
		for i, row := range rows {
			item := map[string]any{}
			for j, c := range m.Columns {
				item[c] = row[j]
			}
			items[i] = item
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"module": m.Path, "site": siteCode, "served_by": hostname, "count": len(items), "items": items})
	}
}
