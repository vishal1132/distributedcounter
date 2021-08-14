package main

import (
	"sync"

	"github.com/vishal1132/crdts/gset"
)

type postLikes struct {
	mu sync.Mutex
	db map[string]gset.Gset
}

func (p *postLikes) AddLike(user, post string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.db[post]
	if !ok {
		v = gset.New()
	}
	v.Append(user)
	p.db[post] = v
	return nil
}

func (p *postLikes) GetLikes(post string) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.db[post]
	if !ok {
		return []string{}, nil
	}
	return v.GetSet(), nil
}

func (p *postLikes) GetLikesCount(post string) (int, error) {
	likes, err := p.GetLikes(post)
	if err != nil {
		return 0, err
	}
	return len(likes), nil
}

func newinmemdb() db {
	return &postLikes{
		db: make(map[string]gset.Gset),
	}
}
