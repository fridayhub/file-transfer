package main

import (
	"encoding/json"
	"fmt"
	"github.com/fridayhub/file-transfer/pool"
	"github.com/fridayhub/file-transfer/util"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"
)

var (
	msgChan  chan util.Msg
	p        pool.Pool
	limitChan  chan struct{}
)


func main() {
	path := "/data/databox/data/image/video/analyse"
	//path := "/home/liujh-h/Pictures"
	var (
		err error
	)
	msgChan = make(chan util.Msg, 5)
	limitChan = make(chan struct{}, 5)
	factory := func()(net.Conn, error) { return net.Dial("tcp", "10.2.73.50:9000") }
	p, err = pool.NewChannelPool(5, 30, factory)
	if err != nil {
		fmt.Printf("pool.NewChannelPool error:%s\n", err)
		return
	}
	go FileToMsg2(path)
	MsgDistribute()
	time.Sleep(3 * time.Second)
	p.Close()
}

func FileToMsg(path string, done chan struct{}) {
	var err error

	info, errf := os.Stat(path)
	if errf != nil {
		fmt.Println("os.Stat errf =", errf)
		return
	}

	if info.IsDir() {
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				fmt.Println("path :", path)
				msgChan <- util.Msg{
					FileName: path,
					Size:     info.Size(),
				}
			}
			return nil
		})
		fmt.Println("eeeeeeeeee:", err)
	} else {
		msgChan <- util.Msg{
			FileName: path,
			Size:     info.Size(),
		}
	}
	if err != nil {
		fmt.Println("send error:", err)
	}
	done <- struct{}{}
}

func FileToMsg2(path string) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Println("os.Stat err:", err)
		return
	}
	if info.IsDir() {
		fs, err := ioutil.ReadDir(path)
		if err != nil {
			fmt.Println("ioutil.ReadDir err:", err)
			return
		}
		for _, file := range fs {
			p := filepath.Join(path, file.Name())
			if file.IsDir() {
				FileToMsg2(p)
			} else {
				fmt.Println("file:", p)
				msgChan <- util.Msg{
					FileName: p,
					Size:     file.Size(),
				}
			}
		}
	} else {
		fmt.Println("file:", path)
		msgChan <- util.Msg{
			FileName: path,
			Size:     info.Size(),
		}
	}
}

func MsgDistribute() {
	for {
		select {
		case msg := <-msgChan:
			limitChan <- struct{}{}
			go Send(msg)
		case <-time.Tick(1 * time.Second):
			return
		}
	}
}

func Send(msg util.Msg) {
	var (
		err error
		n   int
	)
	fmt.Printf("Send start...\n")


	fmt.Println("dddddddddddddddd---1")
	conn, err := p.Get()
	if err != nil {
		fmt.Printf("p.Get error:%s\n", err)
		return
	}
	defer func() {
		<- limitChan
		if err != nil {
			msgChan <- msg
			if pc, ok := conn.(*pool.PoolConn); ok {
				pc.MarkUnusable()
				pc.Close()
			}
		} else {
			defer conn.Close()
		}
	}()
	fmt.Printf("Send get conn, pool len:%d\n", p.Len())

	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("json.Marshal error:%s\n", err)
	}
	n, err = conn.Write(data)
	if err != nil {
		fmt.Println("conn.Write info.Name err:", err)
		return
	}
	fmt.Println("dddddddddddddddd---2")
	buf := make([]byte, 1024)
	n, err = conn.Read(buf)
	if err != nil {
		fmt.Println("conn.Read ok err:", err)
		return
	}
	fmt.Println("dddddddddddddddd---3")
	if "ok" == string(buf[:n]) {
		fmt.Println("dddddddddddddddd---4")
		err = SendFile(msg.FileName, conn)
		fmt.Println("dddddddddddddddd---5")
	} else {
		fmt.Println("send file name error:", string(buf[:n]))
		err = fmt.Errorf("send handshake error")
	}
	fmt.Println("dddddddddddddddd---6")
}

func SendFile(path string, conn net.Conn) error {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("os.Open err:", err)
		return nil
	}
	defer file.Close()
	buf := make([]byte, 1024*2)

	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("eof 文件发送完毕")
				err = nil
			} else {
				fmt.Println("file.Read err :", err)
			}
			break
		}
		if n == 0 {
			fmt.Println("0 文件发送完毕")
			break
		}
		n, err = conn.Write(buf[:n])
		if err != nil {
			fmt.Println("conn.Write error:", err)
			break
		}
	}
	return err
}
