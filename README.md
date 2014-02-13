# riakstore

A session store backend for [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) - [src](https://github.com/gorilla/sessions) using [Riak](http://basho.com).

## Requirements

Depends on the [Riaken](https://github.com/riaken) *riaken-core* Riak library.

## Installation

    go get github.com/boj/riakstore

## Documentation

Available on [godoc.org](http://www.godoc.org/github.com/boj/riakstore).

See http://www.gorillatoolkit.org/pkg/sessions for full documentation on underlying interface.

### Example

    // Fetch new store.
	addrs := []string{"127.0.0.1:8083", "127.0.0.1:8084", "127.0.0.1:8085", "127.0.0.1:8086", "127.0.0.1:8087"}
	store := NewRiakStore(addrs, 5, "sessions", []byte("secret-key"))
    defer store.Close()

    // Get a session.
	session, err := store.Get(req, "session-key")
	if err != nil {
        log.Error(err.Error())
    }

    // Add a value.
    session.Values["foo"] = "bar"

    // Save.
    if err = sessions.Save(req, rsp); err != nil {
        t.Fatalf("Error saving session: %v", err)
    }

    // Delete session.
    session.Options.MaxAge = -1
    if err = sessions.Save(req, rsp); err != nil {
        t.Fatalf("Error saving session: %v", err)
    }

## Notes

See http://docs.basho.com/riak/latest/ops/advanced/backends/multi/ for how to configure multiple backends and bucket level TTL props.

Additional FAQs on TTL:

* http://docs.basho.com/riak/latest/community/faqs/developing/#how-can-i-automatically-expire-a-key-from-riak
* http://docs.basho.com/riak/latest/community/faqs/operations/#what-39-s-the-difference-between-the-riak_kv_cach

