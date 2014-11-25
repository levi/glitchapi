package glitch

import (
	"appengine"
	"appengine/urlfetch"
	"encoding/json"
	"fmt"
	"github.com/levi/twch"
	"net/http"
	"os"
)

var (
	clientKey = os.Getenv("TWITCH_API_KEY")
)

func init() {
	http.HandleFunc("/api/v1/games", gamesHandler)
	http.HandleFunc("/api/v1/streams", streamsHandler)
}

func gamesHandler(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Add("Content-Type", "application/json")

	c := appengine.NewContext(r)
	twitch, err := twitchClient(c)
	if err != nil {
		http.Error(w, "Error creating Twitch connection", http.StatusServiceUnavailable)
		return
	}

	g, _, err := twitch.Games.ListTop(&twch.RequestOptions{})
	if err != nil {
		http.Error(w, "Error fetching games", http.StatusServiceUnavailable)
		return
	}

	games := make([]map[string]interface{}, len(g))
	fmt.Printf("Gameshandler: %d", len(g))
	for k, v := range g {
		games[k] = map[string]interface{}{
			"name":     v.Name,
			"channels": v.Channels,
		}
	}

	b, err := json.Marshal(games)
	if err != nil {
		http.Error(w, "Error marshaling games", http.StatusServiceUnavailable)
		return
	}

	fmt.Fprint(w, string(b))
}

func streamsHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	h := w.Header()
	h.Add("Content-Type", "application/json")

	twitch, err := twitchClient(c)
	if err != nil {
		http.Error(w, "Error creating Twitch connection", http.StatusServiceUnavailable)
		return
	}

	s, _, err := twitch.Streams.ListStreams(&twch.StreamOptions{})
	if err != nil {
		http.Error(w, "Error fetching featured streams", http.StatusServiceUnavailable)
		return
	}

	b, err := json.Marshal(s)
	if err != nil {
		http.Error(w, "Error marshaling featured streams", http.StatusServiceUnavailable)
		return
	}

	fmt.Fprint(w, string(b))
}

func twitchClient(c appengine.Context) (t *twch.Client, err error) {
	client := urlfetch.Client(c)
	return twch.NewClient(clientKey, client)
}
