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
	"github.com/vishal1132/crdts/gset"
)

type server struct {
	// using gset crdt to store likes. This is the in memory database for a service instance.
	postLikes map[string]gset.Gset

	nodeName string

	mu sync.Mutex

	// chanMsg is going to have the likes received from gossip layer.
	chanMsg chan []byte

	broadcast *memberlist.TransmitLimitedQueue
}

func main() {
	s := &server{
		postLikes: make(map[string]gset.Gset),
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
		m.Join(strings.Split(*members, ","))
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
	fmt.Println("http listening on ", l.Addr())
	r := http.NewServeMux()
	{
		r.HandleFunc("/like", s.handlePostLike)
		r.HandleFunc("/likes", s.handleGetLikes)
	}

	srv := &http.Server{
		Handler: r,
	}

	srv.Serve(l)

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

	v, ok := s.postLikes[l.Post]
	if !ok {
		v = gset.New()
	}
	v.Append(l.User)
	s.postLikes[l.Post] = v

	s.broadcast.QueueBroadcast(&l)
	return
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
	v, ok := s.postLikes[post[0]]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "post like this does not exist")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	set := v.GetSet()
	w.WriteHeader(http.StatusOK)
	if len(detail) > 0 {
		fmt.Fprintf(w, strings.Join(set, ","))
		return
	}
	fmt.Fprintf(w, "Likes: %d", len(set))
	return
}
