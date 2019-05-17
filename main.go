package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/k0kubun/pp"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var file, example, driveID, filename string
	flag.StringVar(&file, "config", "", "google drive config file")
	flag.StringVar(&example, "example", "", "google drive example type")
	flag.StringVar(&driveID, "id", "", "target google drive id")
	flag.StringVar(&filename, "file", "", "update file name")
	flag.Parse()

	service, err := DriveSerivce(ctx, file)
	if err != nil {
		log.Println("new derive serivce", err)
		os.Exit(1)
	}

	switch example {
	case "list":
		err := List(ctx, service, driveID)
		if err != nil {
			log.Println("error list google drive files", err)
			os.Exit(1)
		}
	case "list-details":
		err := ListDetails(ctx, service, driveID)
		if err != nil {
			log.Println("error list details google drive files", err)
			os.Exit(1)
		}
	case "list-recursive":
		err := ListRecursive(ctx, service, driveID)
		if err != nil {
			log.Println("error list details google drive files", err)
			os.Exit(1)
		}
	case "update":
		err := Update(ctx, service, driveID, filename)
		if err != nil {
			log.Println("error uddate google drive files", err)
			os.Exit(1)
		}
	}
}

func DriveSerivce(ctx context.Context, config string) (*drive.Service, error) {
	f, err := os.Open(config)
	if err != nil {
		return nil, errors.Wrap(err, "open json config")
	}
	defer f.Close()

	json, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "read file failed")
	}

	jwtConfig, err := google.JWTConfigFromJSON(json)
	if err != nil {
		return nil, errors.Wrap(err, "error new jwt config")
	}

	jwtConfig.Scopes = []string{
		drive.DriveScope,
	}

	cli := jwtConfig.Client(ctx)
	service, err := drive.New(cli)
	if err != nil {
		return nil, errors.Wrap(err, "new drive service")
	}

	return service, nil
}

func List(ctx context.Context, service *drive.Service, driveID string) error {
	list, err := service.Files.List().Q(fmt.Sprintf("'%s' in parents", driveID)).Context(ctx).Do()
	if err != nil {
		return errors.Wrap(err, "list files failed")
	}
	pp.Println("page token:", list.NextPageToken)

	pp.Println("file details")
	for _, f := range list.Files {
		pp.Printf("file name%s data:%+v\n", f.Name, f)
		pp.Printf("file id:%s\nname%s\nparents:%v\n", f.Id, f.Name, f.Parents)
	}
	return nil
}

func ListDetails(ctx context.Context, service *drive.Service, driveID string) error {
	list, err := service.Files.List().Fields("*").Q(fmt.Sprintf("'%s' in parents", driveID)).Context(ctx).Do()
	if err != nil {
		return errors.Wrap(err, "list error")
	}

	pp.Println("file details")
	for _, f := range list.Files {
		pp.Printf("file id:%s\nname%s\nparents:%v\n", f.Id, f.Name, f.Parents)
	}

	return nil
}

func ListRecursive(ctx context.Context, service *drive.Service, driveID string) error {
	list, err := service.Files.List().Fields("files(id, parents, name, mimeType)").Q(fmt.Sprintf("'%s' in parents", driveID)).Context(ctx).Do()
	if err != nil {
		return errors.Wrap(err, "list error")
	}

	pp.Println("token:", list.NextPageToken)

	pp.Println("file details")
	typeFolder := "application/vnd.google-apps.folder"
	for _, f := range list.Files {
		pp.Printf("file id:%s\nname%s\ntype:%v\n", f.Id, f.Name, f.MimeType)
		if f.MimeType == typeFolder {
			err := ListRecursive(ctx, service, f.Id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Update(ctx context.Context, service *drive.Service, driveID, filename string) error {
	list, err := service.Files.List().Fields("*").Q(fmt.Sprintf("'%s' in parents", driveID)).Context(ctx).Do()
	if err != nil {
		return err
	}

	var file *drive.File
	for _, f := range list.Files {
		if strings.Contains(f.Name, filename) {
			file = f
			break
		}
	}

	if file == nil {
		return nil
	}

	year, month, day := time.Now().Date()
	content := &bytes.Buffer{}
	content.WriteString(fmt.Sprintf("%d-%s-%d", year, month.String(), day))

	updatedFile, err := service.Files.Update(file.Id, &drive.File{}).
		Media(content, googleapi.ContentType(file.MimeType)).Context(ctx).Do()
	if err != nil {
		return errors.Wrap(err, "error updated")
	}

	pp.Println("update file", updatedFile.Name)

	return nil
}
