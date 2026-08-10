package oracle

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"

	"kaizengo/internal/module"
)

func init() {
	module.Register(&App{})
}

// App is a playful oracle SPA with a tiny JSON API.
type App struct{}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "oracle",
		Version:     "0.1.0",
		Summary:     "Magic oracle SPA — ask anything",
		Depends:     []string{"core"},
		Installable: true,
	}
}

func (a *App) Setup(host *module.Host) error {
	host.RegisterNav(module.NavEntry{
		ID:        "oracle",
		Title:     "Oracle",
		Route:     "oracle",
		ModuleURL: "/app-assets/oracle/spa.js",
	})
	return nil
}

var answers = []string{
	"Absolutely. The gophers have spoken.",
	"Nope. Not in this timeline.",
	"Ask again after coffee.",
	"The stars say yes — dramatically.",
	"Unclear. Have you tried turning it off and on?",
	"Only if you deploy on a Friday.",
	"Yes, but write a test first.",
	"The oracle is busy reviewing a PR.",
	"Destiny says: ship it.",
	"Hard no. Your future self will thank you.",
	"Perhaps… in Kubernetes.",
	"The answer is 42. You're welcome.",
	"Signs point to a snack break.",
	"Consult the rubber duck, then retry.",
	"Yes — but rename the variable first.",
}

func (a *App) Mount(host *module.Host) error {
	host.Router.Handle(
		"/app-assets/oracle/*",
		http.StripPrefix("/app-assets/oracle/", http.FileServer(http.Dir("apps/oracle/spa"))),
	)

	host.Router.Post("/api/oracle/ask", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Q string `json:"q"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		q := strings.TrimSpace(body.Q)
		if q == "" {
			q = "Will this compile?"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"q":      q,
			"answer": answers[rand.Intn(len(answers))],
		})
	})
	return nil
}
