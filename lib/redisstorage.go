package lib

import (
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

// RidesStorage implements the redis RidesStorage backend for Colly
type RidesStorage struct {
	// Address is the redis server address
	Address string
	// Password is the password for the redis server
	Password string
	// DB is the redis database. Default is 0
	DB int
	// Prefix is an optional string in the keys. It can be used
	// to use one redis database for independent scraping tasks.
	Prefix string
	// Client is the redis connection
	Client *redis.Client

	// Expiration time for Visited keys. After expiration pages
	// are to be visited again.
	Expires time.Duration

	mu sync.RWMutex // Only used for cookie methods.
}

// Init initializes the redis RidesStorage
func (s *RidesStorage) Init() error {
	if s.Client == nil {
		s.Client = redis.NewClient(&redis.Options{
			Addr:     s.Address,
			Password: s.Password,
			DB:       s.DB,
		})
	}
	_, err := s.Client.Ping().Result()
	if err != nil {
		return fmt.Errorf("Redis connection error: %s", err.Error())
	}
	return err
}

// Clear removes all entries from the RidesStorage
func (s *RidesStorage) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.Client.Keys(s.getCookieID("*"))
	keys, err := r.Result()
	if err != nil {
		return err
	}
	r2 := s.Client.Keys(s.Prefix + ":request:*")
	keys2, err := r2.Result()
	if err != nil {
		return err
	}
	keys = append(keys, keys2...)
	keys = append(keys, s.getQueueID())
	return s.Client.Del(keys...).Err()
}

// Visited implements colly/RidesStorage.Visited()
func (s *RidesStorage) Visited(requestID uint64) error {
	return s.Client.Set(s.getIDStr(requestID), "1", s.Expires).Err()
}

// IsVisited implements colly/RidesStorage.IsVisited()
func (s *RidesStorage) IsVisited(requestID uint64) (bool, error) {
	_, err := s.Client.Get(s.getIDStr(requestID)).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// SetCookies implements colly/RidesStorage..SetCookies()
func (s *RidesStorage) SetCookies(u *url.URL, cookies string) {
	// TODO(js) Cookie methods currently have no way to return an error.

	// We need to use a write lock to prevent a race in the db:
	// if two callers set cookies in a very small window of time,
	// it is possible to drop the new cookies from one caller
	// ('last update wins' == best avoided).
	s.mu.Lock()
	defer s.mu.Unlock()
	// return s.Client.Set(s.getCookieID(u.Host), stringify(cnew), 0).Err()
	err := s.Client.Set(s.getCookieID(u.Host), cookies, 0).Err()
	if err != nil {
		// return nil
		log.Printf("SetCookies() .Set error %s", err)
		return
	}
}

// Cookies implements colly/RidesStorage.Cookies()
func (s *RidesStorage) Cookies(u *url.URL) string {
	// TODO(js) Cookie methods currently have no way to return an error.

	s.mu.RLock()
	cookiesStr, err := s.Client.Get(s.getCookieID(u.Host)).Result()
	s.mu.RUnlock()
	if err == redis.Nil {
		cookiesStr = ""
	} else if err != nil {
		// return nil, err
		log.Printf("Cookies() .Get error %s", err)
		return ""
	}
	return cookiesStr
}

// AddRequest implements queue.RidesStorage.AddRequest() function
func (s *RidesStorage) AddRequest(r []byte) error {
	return s.Client.RPush(s.getQueueID(), r).Err()
}

// GetRequest implements queue.RidesStorage.GetRequest() function
func (s *RidesStorage) GetRequest() ([]byte, error) {
	r, err := s.Client.LPop(s.getQueueID()).Bytes()
	if err != nil {
		return nil, err
	}
	return r, err
}

// QueueSize implements queue.RidesStorage.QueueSize() function
func (s *RidesStorage) QueueSize() (int, error) {
	i, err := s.Client.LLen(s.getQueueID()).Result()
	return int(i), err
}

func (s *RidesStorage) getIDStr(ID uint64) string {
	return fmt.Sprintf("%s:request:%d", s.Prefix, ID)
}

func (s *RidesStorage) getCookieID(c string) string {
	return fmt.Sprintf("%s:cookie:%s", s.Prefix, c)
}

func (s *RidesStorage) getQueueID() string {
	return fmt.Sprintf("%s:queue", s.Prefix)
}
