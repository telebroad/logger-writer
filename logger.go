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
	deleteEveryInHour time.Duration
	deleteOlderDays   time.Duration
	writer            chan []byte
	writeLen          chan int
	writerError       chan error
	openedFile        *os.File
	closeFileAfter    time.Duration
	cancelClose       func() bool
	fileLocation      string
}

func New(folder, filePatten string, deleteEveryInHour, deleteOlderDays, closeFileAfter int) *logger {

	l := &logger{
		folder:            folder,
		folderFS:          os.DirFS(folder),
		filePatten:        filePatten,
		deleteEveryInHour: time.Duration(deleteEveryInHour) * time.Hour,
		deleteOlderDays:   time.Hour * 24 * time.Duration(deleteOlderDays),
		closeFileAfter:    time.Duration(closeFileAfter) * time.Second,
		writer:            make(chan []byte, 65535),
		writeLen:          make(chan int, 65535),
		writerError:       make(chan error, 65535),
	}

	go l.writeToFile()
	go l.deleteEvent()
	return l
}

func (l *logger) writeToFile() {

	fileLocation := filepath.Join(l.folder, time.Now().Format(l.filePatten))

	for b := range l.writer {
		// saving to a local variable for the closer func to close this not the old one

		var err error
		fmt.Println(l.openedFile == nil, l.fileLocation != fileLocation)
		if l.openedFile == nil || l.fileLocation != fileLocation {
			l.openedFile, err = os.OpenFile(fileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				l.writeLen <- 0
				l.writerError <- err
			}
			fmt.Println("opened file:", fileLocation)
			l.fileLocation = fileLocation
		} else {
			l.cancelClose()
		}

		fmt.Print(string(b))
		write, err := l.openedFile.Write(b)
		l.writeLen <- write
		l.writerError <- err
		l.cancelClose = l.closerRegister(l.openedFile, fileLocation)
	}
}
func (l *logger) closerRegister(openedFile *os.File, fileLocation string) func() bool {
	return time.AfterFunc(l.closeFileAfter, func() {
		openedFile.Close()
		fmt.Println("closed file:", fileLocation)
		//if the open file is not ben updated with a new file the set it to nil
		if openedFile == l.openedFile {
			l.openedFile = nil
		}
	}).Stop
}
func (l *logger) Write(b []byte) (int, error) {
	l.writer <- b
	return <-l.writeLen, <-l.writerError
}

func (l *logger) deleteEvent() {

	tk := time.NewTicker(l.deleteEveryInHour)
	for range tk.C {
		errList := l.deleteOldFiles(".")
		for _, err := range errList {
			fmt.Println("error deleting files:", err.Error())
		}
	}
}

func (l *logger) deleteOldFiles(currentFolder string) (errList []error) {
	Days30Ago := time.Now().Add(-l.deleteOlderDays)

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
			errL := l.deleteOldFiles(filepath.Join(currentFolder, stat.Name()))
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
