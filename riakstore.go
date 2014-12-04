// Copyright 2014 Brian "bojo" Jones. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package riakstore

import (
	"bytes"
	"encoding/base32"
	"encoding/gob"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	riaken "github.com/riaken/riaken-core"
)

var ErrNoDatabase = errors.New("no databases available")

// Amount of time for cookies/redis keys to expire.
var sessionExpire = 86400 * 30

// RiakStore stores sessions in a redis backend.
type RiakStore struct {
	Riaken        *riaken.Client       // riaken client
	Bucket        string               // bucket to store sessions in
	Codecs        []securecookie.Codec // session codecs
	Options       *sessions.Options    // default configuration
	DefaultMaxAge int                  // default TTL for a MaxAge == 0 session
}

// NewRiakStore returns a new RiakStore.
func NewRiakStore(addrs []string, connections int, bucket string, keyPairs ...[]byte) *RiakStore {
	return &RiakStore{
		Riaken: func() *riaken.Client {
			r := riaken.NewClient(addrs, connections)
			if err := r.Dial(); err != nil {
				panic(err)
			}
			return r
		}(),
		Bucket: bucket,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: sessionExpire,
		},
	}
}

// Close closes the underlying Riaken Client.
func (s *RiakStore) Close() {
	s.Riaken.Close()
}

// Get returns a session for the given name after adding it to the registry.
func (s *RiakStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
func (s *RiakStore) New(r *http.Request, name string) (*sessions.Session, error) {
	var err error
	session := sessions.NewSession(s, name)
	session.Options = &(*s.Options)
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			ok, err := s.load(session)
			session.IsNew = !(err == nil && ok) // not new if no error and data available
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *RiakStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Marked for deletion.
	if session.Options.MaxAge < 0 {
		if err := s.delete(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
	} else {
		// Build an alphanumeric key for the redis store.
		if session.ID == "" {
			session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
		}
		if err := s.save(session); err != nil {
			return err
		}
		encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
		if err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	}
	return nil
}

// save stores the session in riak.
func (s *RiakStore) save(session *sessions.Session) error {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(session.Values)
	if err != nil {
		return err
	}
	rs := s.Riaken.Session()
	if rs == nil {
		return ErrNoDatabase
	}
	defer rs.Release()
	age := session.Options.MaxAge
	if age == 0 {
		age = s.DefaultMaxAge
	}
	b := rs.GetBucket(s.Bucket)
	o := b.Object("session_" + session.ID)
	_, err = o.Store(buf.Bytes())
	return err
}

// load reads the session from riak.
// returns true if there is session data in the DB.
func (s *RiakStore) load(session *sessions.Session) (bool, error) {
	rs := s.Riaken.Session()
	if rs == nil {
		return false, ErrNoDatabase
	}
	defer rs.Release()
	b := rs.GetBucket(s.Bucket)
	o := b.Object("session_" + session.ID)
	data, err := o.Fetch()
	if err != nil {
		return false, err
	}
	if len(data.GetContent()) == 0 {
		return false, nil // no data was associated with this key
	}
	dec := gob.NewDecoder(bytes.NewBuffer(data.GetContent()[0].Value))
	return true, dec.Decode(&session.Values)
}

// delete removes keys from riak if MaxAge<0
func (s *RiakStore) delete(session *sessions.Session) error {
	rs := s.Riaken.Session()
	if rs == nil {
		return ErrNoDatabase
	}
	defer rs.Release()
	b := rs.GetBucket(s.Bucket)
	o := b.Object("session_" + session.ID)
	_, err := o.Delete()
	return err
}
