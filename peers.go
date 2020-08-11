package gocache

// interface that must be implemented to locate 
// the peer taht owns a specific key.
type PeerPicker interface{
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// this interface must be implemented by a peer
type PeerGetter interface{
	Get(group string, key string) ([]byte, error)
}