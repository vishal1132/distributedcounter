package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/memberlist"
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
}

func main() {
	s := &server{
		postLikes: newinmemdb(),
		chanMsg:   make(chan []byte),
	}
	members := flag.String("members", "", "atleast one known member of the cluster(if more, can be comma separated), if not specified, will initiate a cluster")
	// port := flag.Int("port", 0, "specify the http port")
	flag.Parse()

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
		log.Fatalf("Can't create a memberlist got error %e", err)
	}

	if len(*members) > 0 {
		_, _ = m.Join(strings.Split(*members, ","))
	}

	// handle graceful shutdown for memberlist
	// m.Leave()
	// m.Shutdown()

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
		log.Fatalf("Can't listen on port %d got error %e", port, err)
	}
	fmt.Println("http listening on ", l.Addr())
	r := http.NewServeMux()
	{
		r.HandleFunc("/like", s.handlePostLike)
		r.HandleFunc("/likes", s.handleGetLikes)
	}

	srv := &http.Server{
		Handler: r,
	}

	_ = srv.Serve(l)

	// graceful shutdown for http server
	// srv.Shutdown(context.Background())
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
		fmt.Fprint(w, strings.Join(set, ","))
		return
	}
	likes, _ := s.postLikes.GetLikesCount(post[0])
	fmt.Fprintf(w, "Likes: %d", likes)
}
