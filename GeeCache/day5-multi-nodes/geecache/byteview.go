package geecache

// A ByteView holds an immutable view of bytes.
// 抽象一个只读数据结构 ByteView 用来表示缓存值
type ByteView struct {
	b []byte // b 将会存储真实的缓存值, 选择 byte 类型是为了能够支持任意的数据类型的存储，例如字符串、图片等
}

// Len 方法 返回ByteView的长度
func (v ByteView) Len() int {
	return len(v.b) // Len() int 方法，返回其所占的内存大小。
}

// ByteSlice 方法 返回字节切片数据的一个副本
// b 是只读的，使用 ByteSlice() 方法返回一个拷贝，防止缓存值被外部程序修改。
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String 方法 以字符串的形式返回数据，如有必要会生成一个副本
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}