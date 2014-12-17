package datastore

import (
	"appengine/datastore"
	"github.com/levi/twch"
	"sync"
	"time"
)

type GameEntity struct {
	Key      *datastore.Key
	Contents *Game
	Client   *Client
}

type Game struct {
	Name            string
	GiantbombId     int
	Popularity      int
	Viewers         int
	Channels        int
	BoxTemplateURL  string
	LogoTemplateURL string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type GameService struct {
	Client *Client
}

// New creates an unpersisted game entity from a given twtch.Game
func (s *GameService) New(g twch.Game) *GameEntity {
	game := &Game{
		Name:        *g.Name,
		GiantbombId: *g.GiantbombId,
		Viewers:     *g.Viewers,
		Channels:    *g.Channels,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if g.Box != nil && g.Box.Template != nil {
		game.BoxTemplateURL = *g.Box.Template
	}

	if g.Logo != nil && g.Logo.Template != nil {
		game.LogoTemplateURL = *g.Logo.Template
	}

	if g.Popularity != nil {
		game.Popularity = *g.Popularity
	}

	e := &GameEntity{Contents: game, Client: s.Client}
	return e
}

// StoreGames creates or updates a slice of given twch.Games
func (s *GameService) StoreGames(g []twch.Game) (<-chan *GameEntity, <-chan error) {
	ch := make(chan *GameEntity)
	errc := make(chan error, 1)

	time := time.Now()

	go func() {
		var err error
		var wg sync.WaitGroup
		wg.Add(len(g))

		for _, v := range g {
			go func(game twch.Game) {
				g, err := s.GetByName(*game.Name)
				if err != nil {
					return
				}

				if g != nil {
					g := g.Update(game)
					g.Contents.UpdatedAt = time
					g, err := g.Save()
					if err != nil {
						return
					}
					ch <- g
				} else {
					g := s.New(game)
					g.Contents.CreatedAt = time
					g.Contents.UpdatedAt = time
					g, err := g.Save()
					if err != nil {
						return
					}
					ch <- g
				}
				wg.Done()
			}(v)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		errc <- err
	}()

	return ch, errc
}

// Query returns an ancestory query for games
func (s *GameService) Query() *datastore.Query {
	return datastore.NewQuery("Game")
}

// GetByName fetches a GameEntity from the data store matching the passed name string
func (s *GameService) GetByName(n string) (*GameEntity, error) {
	q := s.Query().Filter("Name =", n).Limit(1)

	var result []Game
	keys, err := q.GetAll(*s.Client.Context, &result)
	if err != nil {
		return nil, err
	}

	if len(result) != 0 {
		return &GameEntity{Key: keys[0], Contents: &result[0], Client: s.Client}, nil
	} else {
		return nil, nil
	}
}

// ResetIdle zeroes the viewer and channel count for all game not updated since the
// passed timestamp
func (s *GameService) ResetIdle(t time.Time) (<-chan *GameEntity, <-chan error) {
	ch := make(chan *GameEntity)
	errc := make(chan error, 1)

	go func() {
		var games []Game
		var err error
		q := s.Query().Filter("UpdatedAt <=", t.Add(-1*time.Minute))
		keys, err := q.GetAll(*s.Client.Context, &games)
		if err != nil {
			errc <- err
			return
		}

		var wg sync.WaitGroup
		wg.Add(len(games))
		for i, v := range games {
			go func(g Game, k *datastore.Key) {
				ge := &GameEntity{Key: k, Contents: &g, Client: s.Client}
				ge.Contents.Viewers = 0
				ge.Contents.Channels = 0
				ge.Contents.UpdatedAt = t
				ge, err := ge.Save()
				if err != nil {
					return
				}
				ch <- ge
				wg.Done()
			}(v, keys[i])
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		errc <- err
	}()

	return ch, errc
}

// Update updates the GameEntity's Contents with the passed twch.Game values
// The results are not persisted to the data store. Call Save() after to persist.
func (g *GameEntity) Update(v twch.Game) *GameEntity {
	g.Contents.Viewers = *v.Viewers
	g.Contents.Channels = *v.Channels
	g.Contents.UpdatedAt = time.Now()

	if v.Box != nil && v.Box.Template != nil {
		g.Contents.BoxTemplateURL = *v.Box.Template
	}

	if v.Logo != nil && v.Logo.Template != nil {
		g.Contents.LogoTemplateURL = *v.Logo.Template
	}

	if v.Popularity != nil {
		g.Contents.Popularity = *v.Popularity
	}

	return g
}

// Save persists a GameEntity to the data store
func (g *GameEntity) Save() (*GameEntity, error) {
	key := g.Key
	if key == nil {
		key = datastore.NewIncompleteKey(*g.Client.Context, "Game", nil)
	}
	key, err := datastore.Put(*g.Client.Context, key, g.Contents)
	if err != nil {
		return g, err
	}
	g.Key = key
	return g, nil
}
