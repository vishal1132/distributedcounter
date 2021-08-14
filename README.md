# Distributed Likes Service
Hello world of distributed systems. Yes, we are talking about distributed counter. :stuck_out_tongue: 
Attempt to write the distributed counter in go using CRDTs, because otherwise it would be even hard***er***. This repo is just for learning purpose and not as a production ready code, but will try to achieve as close to it as possible. Plan to use this repository to learn multiple tools and ideologies such as distributed tracing, observability etc.

CRDTs are conflict free replicated data types, and this is an eventual consistent system. Where a like just landed on a post might not reflect to everyone instantly but will appear eventually. This uses gset CRDT which is grow only set written sometime before by me [here](https://github.com/vishal1132/crdts/tree/master/gset) also in go. So you must have already guessed it, yep it doesn't allow you to take back your like or dislike.

Use makefile to run the kubernetes cluster locally, considering you have minikube installed on your local system. Ships the `dockerfile` with the repo by default. The default goal for makefile is help which returns something like this-
![make help](https://i.ibb.co/tQNLhhv/Screenshot-2021-08-14-at-11-52-45-PM.png)

To run the cluster use-
```sh
make run
```
then use `minikube service distributed-likes-service-cluster` to get an IP for your likes cluster service. It will open a url in the browser by default, that url endpoint sort of acts like heartbeat but returns the node name as well in the response, so you can track every request doesn't land to the same node.

Also the endpoints to register the like and get the likes are 
```
curl -XPOST -H "Content-Type:application/json" -d '{"post":"abcd","user":"some"}' /like
curl -XGET likes\?post=abcd
curl -XGET likes\?post=abcd\&detail=true
```
In the example below if you see the request to like landed up at some worker but quickly got propagated and your get request can land at any of the pod afterwards, for example- some other worker or master and you get the updated likes.

![example](https://i.ibb.co/ZX8zhSW/Screenshot-2021-08-14-at-11-59-34-PM.png)

## Progress

- [x] Working
- [x] Orchestrated
- [ ] Globally Available Git Hooks(Lefthook)
- [ ] Distributed Tracing
- [ ] Unit Tests
- [ ] Integration Tests
- [ ] Helm Charts
