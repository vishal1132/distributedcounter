package main

type db interface {
	AddLike(user, post string) error
	GetLikes(post string) ([]string, error)
	GetLikesCount(post string) (int, error)
}
