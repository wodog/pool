package pool

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	config := Config{
		InitialCap: 5,
		MaxCap:     10,
		Timeout:    2 * time.Second,
		Factory: func() (io.Closer, error) {
			return net.Dial("tcp", "baidu.com:80")
		},
	}

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(p.Len())
	ch := time.After(3 * time.Second)
	_, ok := <-ch
	fmt.Println(ok)

	go func() {
		c, err := p.Acquire()
		if err != nil {
			t.Fatal(err)
		}
		p.Release(c)
	}()
	go func() {
		c, err := p.Acquire()
		if err != nil {
			t.Fatal(err)
		}
		p.Release(c)
	}()

	time.Sleep(1 * time.Second)
	fmt.Println(p.Len())
}
