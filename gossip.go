package main

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/memberlist"
	"github.com/vishal1132/crdts/gset"
)

// static check if LikeGossip implements membership.Delegate interface because subscribing server to gossip layer of hashicorp/memberlist protocol
var _ memberlist.Delegate = (*server)(nil)

func (s *server) NodeMeta(limit int) []byte {
	return []byte{}
}

func (s *server) NotifyMsg(b []byte) {
	s.chanMsg <- b
}

func (s *server) GetBroadcasts(overhead, limit int) [][]byte {
	return s.broadcast.GetBroadcasts(overhead, limit)
}

func (s *server) LocalState(join bool) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(s.postLikes)
	return b
}

func (s *server) MergeRemoteState(buf []byte, join bool) {
	if len(buf) == 0 || !join {
		return
	}
	var temp = map[string]gset.Gset{}
	_ = json.Unmarshal(buf, &temp) // deliberately ignore error so linter doesn't complain.
	// converge this temp into s.postLikes
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, value := range temp {
		for _, user := range value.GetSet() {
			_ = s.postLikes.AddLike(key, user)
		}
	}
}

// listening to join and leave events.
func (s *server) NotifyJoin(n *memberlist.Node) {
	fmt.Printf("node joined: %s  address: %s \n", n.String(), n.Addr.String())
}

func (s *server) NotifyLeave(n *memberlist.Node) {
	fmt.Printf("node left: %s \n", n.String())
}

func (s *server) NotifyUpdate(n *memberlist.Node) {
	fmt.Printf("node updated: %s \n", n.String())
}

// implementing broadcast interface.
func (l *likeDomain) Invalidates(previous memberlist.Broadcast) bool {
	return false
}

func (l *likeDomain) Message() []byte {
	b, _ := json.Marshal(l)
	return b
}

func (l *likeDomain) Finished() {
	// to notify the client that the broadcast is finished.
	close(l.chanNotify)
}
