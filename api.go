package glitch

import (
	"appengine"
	"appengine/urlfetch"
	"cache"
	"encoding/json"
	"fmt"
	"github.com/levi/twch"
	"html"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	clientKey = os.Getenv("TWITCH_API_KEY")
)

func init() {
	http.HandleFunc("/api/v1/games", gamesHandler)
	http.HandleFunc("/api/v1/streams", streamsHandler)
}

func gamesHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	result, err := cache.Fetch(c, "top_games", func() (b []byte, err error) {
		twitch, err := twitchClient(c)
		if err != nil {
			return nil, err
		}

		g, _, err := twitch.Games.ListTop(&twch.RequestOptions{ListOptions: twch.ListOptions{Limit: 50}})
		if err != nil {
			return nil, err
		}

		games := make([]map[string]interface{}, len(g))
		fmt.Printf("Games handler: %d", len(g))
		for k, v := range g {
			games[k] = map[string]interface{}{
				"name":           v.Name,
				"boxTemplateURL": v.Box.Template,
			}
		}

		b, err = json.Marshal(games)
		if err != nil {
			return nil, err
		}

		return b, nil
	})

	if err != nil {
		shortError := fmt.Sprintf("Error: %v", err)
		http.Error(w, shortError, http.StatusServiceUnavailable)
		return
	}

	addHTTPHeader(&w, &result.Expiration)

	fmt.Fprint(w, string(result.Value))
}

func streamsHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	r.ParseForm()
	g := r.FormValue("game")
	if g == "" {
		// TODO(levi): Handle error response with JSON output
		http.Error(w, "No game specified", http.StatusNotAcceptable)
		return
	}

	key := fmt.Sprintf("streams_%s", html.EscapeString(g))
	result, err := cache.Fetch(c, key, func() (b []byte, err error) {
		twitch, err := twitchClient(c)
		if err != nil {
			return
		}

		opts := &twch.StreamOptions{
			Game: g,
		}

		s, _, err := twitch.Streams.ListStreams(opts)
		if err != nil {
			return
		}

		b, err = json.Marshal(s)
		if err != nil {
			return
		}

		return
	})

	if err != nil {
		shortError := fmt.Sprintf("Error: %v", err)
		http.Error(w, shortError, http.StatusServiceUnavailable)
		return
	}

	addHTTPHeader(&w, nil)

	fmt.Fprint(w, string(result.Value))
}

func twitchClient(c appengine.Context) (t *twch.Client, err error) {
	client := urlfetch.Client(c)
	return twch.NewClient(clientKey, client)
}

func addHTTPHeader(w *http.ResponseWriter, expiration *time.Time) {
	h := (*w).Header()
	h.Add("Content-Type", "application/json")
	if expiration != nil {
		h.Add("max-age", strconv.FormatInt(int64((*expiration).Sub(time.Now())/time.Second), 10))
		h.Add("Expires", (*expiration).Format(time.RFC1123))
	}
}
