package cache

import (
	"appengine"
	"appengine/memcache"
	"bytes"
	"encoding/gob"
	"time"
)

// Item represents the stored structure inside a memcache value
// Expiration is included in the structure since memcache.Get does not return
// a structure with an expiration time
type Item struct {
	Expiration time.Time
	Value      []byte
}

type Fallback func() (buff []byte, err error)

// Fetch hits cache for the specified key, otherwise invokes its callback and stores the result in the key
func Fetch(c appengine.Context, key string, fallback Fallback) (item *Item, err error) {
	expiration := 15 * time.Minute // TODO(levi): Allow custom expiration to be included
	result, err := memcache.Get(c, key)
	if err != nil && err != memcache.ErrCacheMiss {
		return
	} else if err == nil {
		decBuff := bytes.NewBuffer(result.Value)
		item = new(Item)
		err = gob.NewDecoder(decBuff).Decode(item)
		if err != nil {
			return
		}
		return
	}

	c.Infof("Memcache: Miss - Key=%s\n", key)
	buff, err := fallback()
	if err != nil {
		return
	}

	i := &Item{Expiration: time.Now().Add(expiration), Value: buff}
	encBuff := new(bytes.Buffer)
	err = gob.NewEncoder(encBuff).Encode(i)
	if err != nil {
		return
	}

	err = memcache.Set(c, &memcache.Item{
		Key:        key,
		Value:      encBuff.Bytes(),
		Expiration: expiration,
	})
	if err != nil {
		return
	}

	return i, nil
}
