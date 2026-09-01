package file

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"

	"sms-automation/src/internal/database"
	"sms-automation/src/internal/sms"
)

type File struct {
	Path string
	File os.File
}

func New(path string) *File {
	return &File{
		Path: path,
	}
}

func (f *File) Open() {
	file, err := os.Open(f.Path)

	if err != nil {
		log.Fatal(err)
	}

	f.File = *file
}

func (f *File) Close() {
	f.File.Close()
}

func (f *File) Process() {
	fileRead := bufio.NewScanner(&f.File)

	db, err := database.New()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if fileRead.Scan() {
		//
	}

	for fileRead.Scan() {
		s := sms.Normalize(fileRead.Text())

		fields := strings.Split(s, ",")

		if fields[len(fields)-1] == "" {
			fields = fields[:len(fields)-1]
		}

		data := map[string]any{
			"client":     fields[0],
			"sms_type":   fields[1],
			"operator":   fields[2],
			"code":       fields[3],
			"amount":     fields[4],
			"value":      fields[5],
			"created_at": time.Now(),
		}

		if len(fields) >= 7 {
			data["filial_code"] = fields[6]
		} else {
			data["filial_code"] = nil
		}

		e := db.Table("sms_invicta_2").Create(data)
		if e != nil {
			log.Println(e, s)
			break
		}
	}
}
