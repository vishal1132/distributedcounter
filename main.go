package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/memberlist"
	"github.com/rs/zerolog"
	uuid "github.com/satori/go.uuid"
)

type server struct {
	// postLikes is an interface you can plug your own db implementation, right now using the implementation from inmemdb.go
	postLikes db

	nodeName string

	mu sync.Mutex

	// chanMsg is going to have the likes received from gossip layer.
	chanMsg chan []byte

	broadcast *memberlist.TransmitLimitedQueue

	logger zerolog.Logger
}

func main() {
	s := &server{
		postLikes: newinmemdb(),
		chanMsg:   make(chan []byte),
		logger:    zerolog.New(os.Stdout).With().Timestamp().Logger(),
	}
	members := os.Getenv("members")

	c := memberlist.DefaultLocalConfig()
	{
		hostname, _ := os.Hostname()
		c.Delegate = s
		c.Events = s
		c.BindPort = 9293
		c.Name = hostname + ":" + uuid.NewV4().String()
	}

	m, err := memberlist.Create(c)
	if err != nil {
		// this is the core of the application. So it makes sense to fatal the program with err here.
		s.logger.Fatal().Msgf("Can't create a memberlist got error %e", err)
	}

	if len(members) > 0 {
		// members are only going to be specified for worker nodes.
		member := fmt.Sprintf("%s:%s", strings.Split(members, ",")[0], os.Getenv("membership_port"))
		_, err := m.Join([]string{member})
		if err != nil {
			s.logger.Fatal().Msgf("can't join membership cluster while in worker mode got error %e", err)
		}
	}

	// handle graceful shutdown for memberlist
	// m.Leave()
	// m.Shutdown()

	go s.loopSM()

	s.nodeName = c.Name
	s.broadcast = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return m.NumMembers()
		},
		// Max number of retransmissions for a message before we give up.
		RetransmitMult: 4,
	}
	port := 8080
	l, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		s.logger.Fatal().Msgf("Can't listen on port %d got error %e", port, err)
	}

	s.logger.Info().Msgf("Node %s listening on %s", s.nodeName, l.Addr().String())
	r := http.NewServeMux()
	{
		r.HandleFunc("/", s.logMiddleware(s.handleHeartBeat))
		r.HandleFunc("/like", s.logMiddleware(s.handlePostLike))
		r.HandleFunc("/likes", s.logMiddleware(s.handleGetLikes))
	}

	_ = http.Serve(l, r) // l.Close()

	// _ = srv.Serve(l)

	// graceful shutdown for http server
	// srv.Shutdown(context.Background())
}

func (s *server) handleHeartBeat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "{\"node_name\":\""+s.nodeName+"\"}")
}

func (s *server) logMiddleware(next http.HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug().Msgf("Request landed on %s %s %s %s", s.nodeName, r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	}
}

func (s *server) loopSM() {
	for i := range s.chanMsg {
		s.logger.Debug().Msgf("Received message %s", string(i))
		var l likeDomain
		err := json.Unmarshal(i, &l)
		if err != nil {
			s.logger.Debug().Msgf("Error unmarshalling message %e", err)
			continue
		}
		// if error is not nil add that to gset.
		_ = s.postLikes.AddLike(l.User, l.Post)
	}
}

type likeDomain struct {
	Post       string `json:"post"`
	User       string `json:"user"`
	chanNotify chan struct{}
}

func (s *server) handlePostLike(w http.ResponseWriter, r *http.Request) {
	// request should look something like {"post":"post-name","like":"username"}
	// "post":"post-name" is the name of the post.
	// "like":"username" is the like from the user username.
	b, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Error reading request body %s", err)
		return
	}

	l := likeDomain{}
	if err := json.Unmarshal(b, &l); err != nil {
		fmt.Fprintf(w, "Error reading request body %s", err)
		return
	}

	_ = s.postLikes.AddLike(l.User, l.Post)

	s.broadcast.QueueBroadcast(&l)
	fmt.Fprintf(w, "Successfully wrote to %s", s.nodeName)
}

func (s *server) handleGetLikes(w http.ResponseWriter, r *http.Request) {
	// get the likes from the in memory database.
	post := r.URL.Query()["post"]
	detail := r.URL.Query()["detail"]
	if len(post) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "bad request required post query parameter")
		return
	}

	w.WriteHeader(http.StatusOK)
	if len(detail) > 0 {
		set, _ := s.postLikes.GetLikes(post[0])
		fmt.Fprintf(w, "Server %s returned %s", s.nodeName, strings.Join(set, ","))
		return
	}
	likes, _ := s.postLikes.GetLikesCount(post[0])
	fmt.Fprintf(w, "Server %s returned Likes: %d", s.nodeName, likes)
}
