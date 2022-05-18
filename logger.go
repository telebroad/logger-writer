package logger

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type logger struct {
	folder            string
	folderFS          fs.FS
	filePatten        string
	deleteEveryInHour int
	deleteOlderDays   int
	writer            chan []byte
	writeLen          chan int
	writerError       chan error
	openedFile        *os.File
	timeout           *time.Timer
}

func New(folder, filePatten string, deleteEveryInHour, deleteOlderDays int) *logger {

	l := &logger{
		folder:            folder,
		folderFS:          os.DirFS(folder),
		filePatten:        filePatten,
		deleteEveryInHour: deleteEveryInHour,
		deleteOlderDays:   deleteOlderDays,
		writer:            make(chan []byte),
	}

	go l.writeToFile()
	return l
}

func (l *logger) writeToFile() {
	for b := range l.writer {
		var err error
		l.openedFile, err = os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			l.writeLen <- 0
			l.writerError <- err
		}
		write, err := l.openedFile.Write(b)
		l.writeLen <- write
		l.writerError <- err
		//l.filePatten
	}
}
func (l *logger) End() {
	l.timeout = time.NewTimer(time.Second * 2)
	<-l.timeout.C

}
func (l *logger) Write(b []byte) (int, error) {
	l.writer <- b
	return <-l.writeLen, <-l.writerError
}

func (l *logger) DeleteEvent() {
	tk := time.NewTicker(time.Duration(l.deleteEveryInHour) * time.Hour)
	for range tk.C {
		errList := l.DeleteOldFiles(".")
		for _, err := range errList {
			fmt.Println("error deleting files:", err.Error())
		}
	}
}

func (l *logger) DeleteOldFiles(currentFolder string) (errList []error) {
	Days30Ago := time.Now().Add(-(time.Hour * 24 * time.Duration(l.deleteOlderDays)))

	fs.WalkDir(l.folderFS, currentFolder, func(path string, d fs.DirEntry, err error) error {
		// getting s.DirEntry.Info
		stat, err := d.Info()
		if err != nil {
			return err
		}
		// checking if it is the current dir
		if stat.Name() == currentFolder {
			return nil
		}
		//
		if stat.IsDir() {
			errL := l.DeleteOldFiles(filepath.Join(currentFolder, stat.Name()))
			if len(errL) != 0 {
				errList = append(errList, errL...)
				return nil
			}
		}
		// if the file or empty folder is older than Days30Ago delete it
		if stat.ModTime().Before(Days30Ago) {
			file := filepath.Join(l.folder, currentFolder, path)
			err := os.Remove(file)
			if err == nil {
				fmt.Println("successfully deleted", file)
				return nil
			}
		}

		return nil
	})
	return
}
