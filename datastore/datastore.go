package datastore

import (
	"appengine"
)

type Client struct {
	Games   *GameService
	Context *appengine.Context
}

func NewClient(c *appengine.Context) *Client {
	cl := new(Client)
	cl.Games = &GameService{Client: cl}
	cl.Context = c
	return cl
}
