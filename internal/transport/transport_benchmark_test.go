package transport

import (
	"context"
	"crypto/tls"
	"testing"
)

func BenchmarkWSConnWriteSmall(b *testing.B) {
	wsConn := &WSConn{}
	data := []byte("small test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wsConn.Write(data)
	}
}

func BenchmarkWSConnWriteMedium(b *testing.B) {
	wsConn := &WSConn{}
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wsConn.Write(data)
	}
}

func BenchmarkWSConnWriteLarge(b *testing.B) {
	wsConn := &WSConn{}
	data := make([]byte, 65536)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wsConn.Write(data)
	}
}

func BenchmarkDialTCP(b *testing.B) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Dial(ctx, "127.0.0.1:1", tlsConfig, false)
	}
}
