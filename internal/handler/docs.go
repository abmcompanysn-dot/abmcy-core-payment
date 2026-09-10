package handler

import (
	_ "embed"
	"net/http"
)

//go:embed integration.md
var integrationMD string

// IntegrationDoc — GET /docs/integration.md : la fiche d'intégration en
// Markdown brut. Source unique (le fichier INTEGRATION.md du repo, copié à
// la racine du contexte de build sous integration.md). Sert au rendu dans
// la console ET au téléchargement.
func (h *PaymentHandler) IntegrationDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if r.URL.Query().Get("download") != "" {
		w.Header().Set("Content-Disposition", `attachment; filename="ABMCY-Core-Integration.md"`)
	}
	_, _ = w.Write([]byte(integrationMD))
}
