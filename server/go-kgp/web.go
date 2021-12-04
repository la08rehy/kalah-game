package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"
)

//go:embed html
var html embed.FS

var T *template.Template
var funcs = template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
	"dec": func(i int) int {
		return i - 1
	},
	"isOver": func(g Game) bool {
		return g.IsOver()
	},
	"timefmt": func(t time.Time) string {
		return t.Format(time.RFC822)
	},
	"result": func(out Outcome) string {
		switch out {
		case ONGOING:
			return "Ongoing"
		case WIN:
			return "South won"
		case DRAW:
			return "Draw"
		case LOSS:
			return "North won"
		case RESIGN:
			return "Resignation"
		default:
			return "???"
		}
	},
}

func init() {
	staticfs, err := fs.Sub(html, "html/static")
	if err != nil {
		log.Fatal(err)
	}
	static := http.FileServer(http.FS(staticfs))

	// Install HTTP handlers
	http.HandleFunc("/games", listGames)
	http.HandleFunc("/agents", listAgents)
	http.HandleFunc("/game/", showGame)
	http.HandleFunc("/agent/", showAgent)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			err := T.ExecuteTemplate(w, "index.tmpl", struct{}{})
			if err != nil {
				log.Print(err)
			}

		case "/about":
			err := T.ExecuteTemplate(w, "about.tmpl", struct{}{})
			if err != nil {
				log.Print(err)
			}
		default:
			static.ServeHTTP(w, r)
		}
	})

	// Parse templates
	T = template.Must(template.New("").Funcs(funcs).ParseFS(html, "html/*.tmpl"))

}

func showGame(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	c := make(chan *Game)
	dbact <- queryGame(id, c)
	err = T.ExecuteTemplate(w, "show-game.tmpl", <-c)
	if err != nil {
		log.Print(err)
	}
}

func showAgent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}

	c := make(chan *Agent)
	dbact <- queryAgent(id, c)
	games := make(chan *Game)
	dbact <- queryGames(games, page-1, &id)

	err = T.ExecuteTemplate(w, "show-agent.tmpl", struct {
		Agent chan *Agent
		Games chan *Game
		Page  int
	}{c, games, page})
	if err != nil {
		log.Print(err)
	}
}

func listGames(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}

	c := make(chan *Game)
	dbact <- queryGames(c, page-1, nil)
	err = T.ExecuteTemplate(w, "list-games.tmpl", struct {
		Games chan *Game
		Page  int
	}{c, page})
	if err != nil {
		log.Print(err)
	}
}

func listAgents(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}

	c := make(chan *Agent)
	dbact <- queryAgents(c, page-1)
	err = T.ExecuteTemplate(w, "list-agents.tmpl", struct {
		Agents chan *Agent
		Page   int
	}{c, page})
	if err != nil {
		log.Print(err)
	}
}