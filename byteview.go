package gocache


// ByteView immutable view of bytes
type ByteView struct{
	b []byte
}

func (v ByteView) Len() int64{
	return int64(len(v.b))
}

func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}


func (v ByteView) String() string{
	return string(v.b)
}

func cloneBytes(b []byte) []byte{
	c:= make([]byte,len(b))
	copy(c,b)
	return c
}