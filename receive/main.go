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

const BASEPATH = "./"

func main() {
	addr := ":9000"
	listenner, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("net.Listen err =", err)
		return
	}
	defer listenner.Close()
	for {
		conn, errl := listenner.Accept()
		if errl != nil {
			fmt.Println("listenner.Accept err =", errl)
			return
		}
		go func(conn net.Conn) {
			var (
				n   int
				msg util.Msg
			)
			buf := make([]byte, 1024)
			for {
				n, err = conn.Read(buf)
				if err != nil {
					fmt.Println("conn.Read fileName err =", err)
					return
				}
				err = json.Unmarshal(buf[:n], &msg)
				if err != nil {
					n, err = conn.Write([]byte("error"))
					return
				}

				n, err = conn.Write([]byte("ok"))
				if err != nil {
					fmt.Println("conn.Write ok err =", err)
					return
				}
				msg.FileName = filepath.Join(BASEPATH, msg.FileName)
				if err = CheckPath(filepath.Dir(msg.FileName)); err != nil {
					fmt.Println(err)
					return
				}
				if err = RecvFile(msg.FileName, conn, msg.Size); err != nil {
					fmt.Printf("RecvFile err:%s\n", err)
					return
				}
			}

		}(conn)
	}
}

func CheckPath(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func RecvFile(fileName string, conn net.Conn, size int64) error {
	var (
		count int64
		block int
		n     int
		err   error
		buf   []byte
	)
	count = 0
	block = 1024 * 4

	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("os.Create err =", err)
		return err
	}

	defer file.Close()

	buf = make([]byte, block)
	for {
		remain := size - count
		if remain >= int64(block) {
			n, err = conn.Read(buf)
		} else {
			buf = make([]byte, remain)
			n, err = conn.Read(buf)
		}
		if err != nil {
			if err == io.EOF {
				fmt.Println("任务完成")
			} else {
				fmt.Println("conn.Read err =", err)
			}
			return err
		}

		n, err = file.Write(buf[:n])
		if err != nil {
			fmt.Println("file.Write err =", err)
			break
		}
		count += int64(n)
		fmt.Printf("count:%d,size:%d\n", count, size)
		if count == size {
			fmt.Printf("文件：%s, 接受完成\n", fileName)
			return nil
		}
	}
	return err
}
