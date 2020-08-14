package gocache

import pb "./gocachepb"

// interface that must be implemented to locate 
// the peer taht owns a specific key.
// 其实就是的http节点选择的抽象类
type PeerPicker interface{
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// this interface must be implemented by a peer
type PeerGetter interface{
	Get(in *pb.Request, out *pb.Response) error
	//Get(group string, key string) ([]byte, error)
}