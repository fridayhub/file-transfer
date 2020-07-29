package main

import (
	"encoding/json"
	"file-transfer/util"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

//var (
//	fileChan chan string
//)

func main() {
	addr := ":9000"
	path := "/home/liujh-h/Pictures"

	conn, err := net.Dial("tcp", addr)
	if err != nil{
		fmt.Println("net.Dial err =",err)
		return
	}
	defer conn.Close()
	//fileChan = make(chan string, 1000)

	ListAndSend(conn, path)
}

func ListAndSend(conn net.Conn, path string)  {
	var err error

	info, errf := os.Stat(path)
	if errf != nil{
		fmt.Println("os.Stat errf =", errf)
		return
	}

	if info.IsDir() {
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			fmt.Println("path:", path)
			if !info.IsDir() {
				if err = Send(conn, util.Msg{
					FileName: path,
					Size:     info.Size(),
				}); err != nil {
					return err
				}
			}
			return nil
		})
	} else {
		err = Send(conn, util.Msg{
			FileName: path,
			Size:     info.Size(),
		})
	}
	if err != nil {
		fmt.Println("send error:", err)
	}
}

func Send(conn net.Conn, msg util.Msg) error {
	fmt.Println("........:", msg)
	var err error
	data, err := json.Marshal(msg)
	_, err = conn.Write(data)
	if err != nil{
		fmt.Println("conn.Write info.Name err =",err)
		return err
	}

	var n int
	buf := make([]byte, 1024)
	n, err = conn.Read(buf)
	if err != nil{
		fmt.Println("conn.Read ok err =", err)
		return err
	}

	if "ok" == string(buf[:n]){
		fmt.Println("ok")
		SendFile(msg.FileName, conn)
	}
	return nil
}

func SendFile(path string, conn net.Conn){
	file , err := os.Open(path)

	if err != nil{
		fmt.Println("os.Open err =", err)
		return
	}
	defer file.Close()
	buf := make([]byte, 1024 * 2)

	for {
		n, err := file.Read(buf)

		if err != nil{
			if err == io.EOF{
				fmt.Println("文件发送完毕")
			} else{
				fmt.Println("file.Read err =",err)
			}
			return
		}
		if n == 0{
			fmt.Println("文件发送完毕")
			break
		}
		n, err = conn.Write(buf[:n])
		if err != nil {
			fmt.Println("conn.Write error:", err)
			return
		}
	}
}
