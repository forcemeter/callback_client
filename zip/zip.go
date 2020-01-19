package zip

import (
    "bytes"
    "compress/zlib"
    "fmt"
    "io"
    "os"
)

//进行zlib压缩
func Zip(src []byte) []byte {
    var in bytes.Buffer
    w := zlib.NewWriter(&in)
    w.Write(src)
    w.Close()
    return in.Bytes()
}

//进行zlib解压缩
func Unzip(compressSrc []byte) []byte {
    b := bytes.NewReader(compressSrc)
    var out bytes.Buffer
    r, _ := zlib.NewReader(b)
    io.Copy(&out, r)
    return out.Bytes()
}