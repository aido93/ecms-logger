package ECMSLogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var maxSize int64

func rotateLogDir(reserve *Reserve) error {
	if reserve == nil {
		return nil
	}
	var files []string
	err := filepath.Walk(reserve.Dir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".log" {
			files = append(files, info.Name())
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, f := range files {
		oldpath := path.Join(reserve.Dir, f)
		a := strings.Split(f, "_")
		tmp, _ := strconv.Atoi(a[0])
		tmp += 1
		if tmp > reserve.Rotate.MaxFiles {
			err := os.Remove(oldpath)
			if err != nil {
				return err
			}
			continue
		}
		a[0] = strconv.Itoa(tmp)
		fn := strings.Join(a, "_")
		newpath := path.Join(reserve.Dir, fn)
		err := os.Rename(oldpath, newpath)
		if err != nil {
			return err
		}
	}
	return nil
}

func flushToDisk(reserve *Reserve, data []byte) error {
	if err := rotateLogDir(reserve); err != nil {
		// Cannot operate in reserve dir
		// Nothing to do below
		return err
	}
	fn := "0_" + strconv.Itoa(int(time.Now().Unix())) + ".log"
	filename := path.Join(reserve.Dir, fn)
	return ioutil.WriteFile(filename, data, 0644)
}

func reserveRecords(logStorage []AccessRecord, reserve *Reserve) {
	if reserve == nil {
		log.Info("Reserving logs is disabled")
		return
	}
	buf := bytes.NewBuffer([]byte{})
	for _, lr := range logStorage {
		prevSize := buf.Len()
		b, _ := json.Marshal(lr)
		if int64(prevSize) < maxSize && int64(prevSize)+int64(len(b)+1) >= maxSize {
			if err := flushToDisk(reserve, buf.Bytes()); err != nil {
				log.Error(err)
				return
			}
			buf.Reset()
		}
		fmt.Fprintln(buf, b)
	}
	if err := flushToDisk(reserve, buf.Bytes()); err != nil {
		log.Error(err)
		return
	}
	buf.Reset()
}
