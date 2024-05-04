package gcloud

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lomorage/lomo-backup/common/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const mimiTypeFolder = "application/vnd.google-apps.folder"

type Config struct {
	CredFilename  string
	TokenFilename string
}

type FileMetadata struct {
	Hash    string
	ModTime time.Time
}

type Client struct {
	srv *drive.Service
}

func CreateClient(conf *Config) (*Client, error) {
	ctx := context.Background()
	b, err := os.ReadFile(conf.CredFilename)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	token, err := tokenFromFile(conf.TokenFilename)
	if err != nil {
		return nil, err
	}

	cli := &Client{}

	cli.srv, err = drive.NewService(ctx, option.WithHTTPClient(config.Client(ctx, token)))
	return cli, err
}

func (c *Client) GetFileID(filename, parentFolderID string) (string, string, error) {
	query := fmt.Sprintf("name = '%s'", filename)
	if parentFolderID != "" {
		query += fmt.Sprintf(" and '%s' in parents", parentFolderID)
	}
	files, err := c.srv.Files.List().Q(query).PageSize(1).Fields("files(id, name, parents)").Do()
	if err != nil {
		return "", "", err
	}

	if len(files.Files) == 0 {
		return "", "", nil
	}
	parentID := ""
	if len(files.Files[0].Parents) == 0 {
		logrus.Warnf("File '%s' is not in any folder.", filename)
	} else {
		parentID = files.Files[0].Parents[0]
	}
	return files.Files[0].Id, parentID, nil
}

func (c *Client) GetAndCreateFileIDIfNotExist(filename, parentFolderID string, r io.Reader, m FileMetadata) (bool, string, error) {
	fileID, pid, err := c.GetFileID(filename, parentFolderID)
	if err != nil {
		return false, "", err
	}
	if fileID == "" {
		// not exist, create new one
		fileID, err = c.CreateFile(filename, parentFolderID, r, m)
		return false, fileID, err
	}
	if parentFolderID != "" && pid != parentFolderID {
		return false, "", fmt.Errorf("obtain wrong parent ID while attempting to find '%s' folder ID ", filename)
	}
	return true, fileID, nil
}

func (c *Client) CreateFile(filename, parentFolderID string, r io.Reader, m FileMetadata) (string, error) {
	file := &drive.File{
		Name:        filename,
		CreatedTime: m.ModTime.Format(time.RFC3339),
	}
	if m.Hash != "" {
		file.AppProperties = map[string]string{
			"hash": m.Hash,
		}
	}
	if parentFolderID != "" {
		file.Parents = []string{parentFolderID}
	}
	if r == nil {
		// it is a folder
		file.MimeType = mimiTypeFolder
		f, err := c.srv.Files.Create(file).Do()
		if err != nil {
			return "", err
		}
		return f.Id, nil
	}
	f, err := c.srv.Files.Create(file).Media(r).Do()
	if err != nil {
		return "", err
	}
	return f.Id, nil
}

func (c *Client) ListFiles(folderID string) ([]*types.DirInfo, []*types.FileInfo, error) {
	query := fmt.Sprintf("'%s' in parents", folderID)
	list, err := c.srv.Files.List().Q(query).Fields("files(id, name, size, mimeType, createdTime)").Do()
	if err != nil {
		return nil, nil, err
	}

	files := []*types.FileInfo{}
	folders := []*types.DirInfo{}
	for _, f := range list.Files {
		t, err := time.Parse(time.RFC3339, f.CreatedTime)
		if err != nil {
			logrus.Warnf("Parse %s create time: %s", f.Name, err)
		}
		if f.MimeType == mimiTypeFolder {
			folders = append(folders, &types.DirInfo{
				RefID:   f.Id,
				Path:    f.Name,
				ModTime: t,
			})
			continue
		}
		files = append(files, &types.FileInfo{
			RefID:   f.Id,
			Name:    f.Name,
			Size:    int(f.Size),
			ModTime: t,
		})
	}
	return folders, files, nil
}
