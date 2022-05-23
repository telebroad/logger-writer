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

func New(folder, filePatten string, deleteEveryInHour, deleteOlderDays, closeFileAfter int) (*logger, error) {

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

	if _, err := os.Stat(l.folder); !os.IsNotExist(err) {
		err = os.MkdirAll(l.folder, 0777)
		if err != nil {
			err = fmt.Errorf("creating folder error %w", err)
			return l, err
		}
	}

	go l.writeToFile()
	go l.deleteEvent()
	return l, nil

}

func (l *logger) writeToFile() {

	for {
		// saving to a local variable for the closer func to close this not the old one

		select {
		case <-time.After(l.closeFileAfter):
			l.closeFile()
		case b := <-l.writer:
			fileLocation := filepath.Join(l.folder, time.Now().Format(l.filePatten))
			if l.fileLocation != fileLocation {
				l.closeFile()
				l.fileLocation = fileLocation
			}
			if l.openedFile == nil {
				err := l.openFile()
				if err != nil {
					l.writeLen <- 0
					l.writerError <- err
					return
				}
			}

			fmt.Print(string(b))
			write, err := l.openedFile.Write(b)
			l.writeLen <- write
			l.writerError <- err
		}
	}
}

func (l *logger) openFile() (err error) {
	l.openedFile, err = os.OpenFile(l.fileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return
	}
	fmt.Println("opened file:", l.fileLocation)
	return
}

func (l *logger) closeFile() error {
	if l.openedFile == nil {
		return fmt.Errorf("logger.openedFile is nil")
	}
	err := l.openedFile.Close()
	fmt.Println("closing file:", l.fileLocation)
	l.openedFile = nil
	return err
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
