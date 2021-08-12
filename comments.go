// Distributed like service with in memory database.
// Anyone can create a post, and anyone can like a post.
// Request to like might land on any instance of the service running, therefore we will gossip the like to reach every node.
// This is going to be eventually consistent. And getLikes() might not implement serializability.
// For serializability you might want to do something like writing the like to a quorum of nodes and then reading it from the quorum of nodes.
// But this writes just to a single node on which the request lands, and then it's the responsibility of the node to gossip the like to the rest of the nodes.
// Because this does not require stronger consistency, and is just for a practice and using CRDTs.
package main
