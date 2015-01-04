package glitch

import (
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
	"cache"
	"datastore"
	"encoding/json"
	"fmt"
	"github.com/levi/twch"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	clientKey = os.Getenv("TWITCH_API_KEY")
)

func init() {
	http.HandleFunc("/api/v1/app_config", configHandler)
	http.HandleFunc("/api/v1/games", gamesHandler)
	http.HandleFunc("/api/v1/streams", streamsHandler)

	http.HandleFunc("/tasks/fetch/games", fetchGamesHandler)
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	h := w.Header()
	h.Add("Content-Type", "application/json")

	config, err := memcache.Get(c, "app_config")
	if err != nil && err != memcache.ErrCacheMiss {
		shortError := fmt.Sprintf("Error: %v", err)
		http.Error(w, shortError, http.StatusServiceUnavailable)
		return
	}

	if err == nil {
		fmt.Fprint(w, string(config.Value))
		return
	}

	dir := "./scripts"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		shortError := fmt.Sprintf("Error: %v", err)
		http.Error(w, shortError, http.StatusServiceUnavailable)
		return
	}

	scripts := make(map[string]string)
	for _, f := range files {
		n := f.Name()
		script, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, n))
		if err != nil {
			return
		}
		sn := strings.Replace(n, ".min.js", "", 1)
		scripts[sn] = string(script)
	}

	b, err := json.Marshal(scripts)
	if err != nil {
		shortError := fmt.Sprintf("Error: %v", err)
		http.Error(w, shortError, http.StatusServiceUnavailable)
		return
	}

	config = &memcache.Item{
		Key:   "app_config",
		Value: b,
	}
	if err := memcache.Set(c, config); err != nil {
		shortError := fmt.Sprintf("Error: %v", err)
		http.Error(w, shortError, http.StatusServiceUnavailable)
		return
	}

	fmt.Fprint(w, string(b))
}

func gamesHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	d := datastore.NewClient(&c)

	result, err := cache.Fetch(c, "top_games", func() (b []byte, err error) {
		var games []datastore.Game
		q := d.Games.Query().Order("-Viewers").Limit(50)
		_, err = q.GetAll(c, &games)
		if err != nil {
			return nil, err
		}

		g := make([]map[string]interface{}, len(games))
		for k, v := range games {
			g[k] = map[string]interface{}{
				"name":           v.Name,
				"boxTemplateURL": v.BoxTemplateURL,
			}
		}

		b, err = json.Marshal(g)
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

func fetchGamesHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	d := datastore.NewClient(&c)
	_ = d
	twitch, err := twitchClient(c)
	_ = twitch
	if err != nil {
		c.Errorf("Fetch Games: Error creating twitch client - %s", err)
		return
	}

	var updateTime time.Time
	o := &twch.RequestOptions{ListOptions: twch.ListOptions{Limit: 100}}
	for {
		g, resp, err := twitch.Games.ListTop(o)
		_ = g
		if err != nil {
			http.Error(w, `{ error: "Error fetching twitch games" }`, 503)
			return
		}

		gc, errc := d.Games.StoreGames(g)
		for ge := range gc {
			updateTime = ge.Contents.UpdatedAt
			c.Infof("Fetch Games: Stored game - %s | Viewers: %d | Channels: %d", ge.Contents.Name, ge.Contents.Viewers, ge.Contents.Channels)
		}
		if err := <-errc; err != nil {
			http.Error(w, `{ error: "Fetch Games: Error storing game" }`, 503)
			return
		}

		// NOTE: Breaking after first fetch to reduce db writes for now
		break

		if resp.NextOffset != nil && *resp.NextOffset < *resp.Total {
			c.Infof("Fetch Games: Fetched games %d of %d", *resp.NextOffset, *resp.Total)
			o.Offset = *resp.NextOffset
		} else {
			c.Infof("Fetch Games: Completed")
			break
		}
	}

	gc, errc := d.Games.ResetIdle(updateTime)
	for ge := range gc {
		c.Infof("Fetch Games: Reset idle game - %s", ge.Contents.Name)
	}
	if err := <-errc; err != nil {
		http.Error(w, `{ error: "Fetch Games: Error reseting idle games" }`, 503)
		return
	}

	memcache.Delete(c, "top_games")

	gamesHandler(w, r)
}

func twitchClient(c appengine.Context) (t *twch.Client, err error) {
	client := urlfetch.Client(c)
	return twch.NewClient(clientKey, client)
}

func addHTTPHeader(w *http.ResponseWriter, expiration *time.Time) {
	h := (*w).Header()
	h.Add("Content-Type", "application/json")
	if expiration != nil {
		maxAge := int64((*expiration).Sub(time.Now()) / time.Second)
		expires := (*expiration).Format(time.RFC1123)
		h.Add("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
		h.Add("Expires", strings.Replace(expires, "UTC", "GMT", 1))
	}
}
