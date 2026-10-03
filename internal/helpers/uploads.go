package helpers

import (
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const AVATAR_PATH = "uploads/avatars"
const POSTS_PATH = "uploads/posts"
const COMMENTS_PATH = "uploads/comments"

/*
this function main functionality is to upload the files to the specefic folder, but it also validate the size and the
dimensions of the file using DecodeConfig. Because this function is made to upload images or gif's

Parameters:

	file multipart.File, header *multipart.FileHeader, Type string
															-> Type are strings that point to pathes mentioned in the begening of the file

Returns:

	string
		-> path of the upload
	error
		-> nil if success
*/
func SaveUploads(file multipart.File, header *multipart.FileHeader, Type string) (string, error) {
	if header.Size > 5*1024*1024 {
		return "", errors.New("image must be no larger than 5 MB")
	}

	// we read the file in the handler so we are reading it again. just so we make sure we read it from the begging we use file.Seek with offset 0
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	config, format, err := image.DecodeConfig(file)
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 40_000_000 {
		return "", errors.New("invalid or oversized image")
	}
	extensions := map[string]string{"jpeg": ".jpg", "png": ".png", "gif": ".gif"}
	extension, ok := extensions[format]
	if !ok {
		return "", errors.New("only JPEG, PNG and GIF images are allowed")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	var path string

	if Type == "avatar" {
		path = AVATAR_PATH
	} else if Type == "post" {
		path = POSTS_PATH
	} else if Type == "comment" {
		path = COMMENTS_PATH
	} else {
		return "", os.ErrInvalid
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}

	filename := uuid.New().String() + extension
	filePath := filepath.Join(path, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(filePath)
		return "", err
	}

	if Type == "avatar" {
		return "avatars/" + filename, nil
	}
	if Type == "comment" {
		return "comments/" + filename, nil
	}

	return "posts/" + filename, nil
}

/*
function mostly used after deleteing or changing avatars and it is used to delete the prevoius avatar from the database

Parameters:

	avatarPath string
		-> the path should be relative to the file /uploads/

Return:

	error
		-> nil if success
*/
func DeleteAvatar(avatarPath string) error {
	if avatarPath == "" {
		return nil
	}

	if filepath.Base(avatarPath) == "default.png" {
		return nil
	}

	filePath := filepath.Join("uploads", filepath.FromSlash(avatarPath))

	return os.Remove(filePath)
}

/*
DeleteUpload removes a post or comment picture from the uploads folder.
the path is the one saved in the database, like "posts/abc.png".
it never deletes the default avatar and it only works inside uploads/
*/
func DeleteUpload(path string) {
	if path == "" || filepath.Base(path) == "default.png" {
		return
	}

	clean := filepath.Clean(filepath.FromSlash(path))
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return
	}

	if err := os.Remove(filepath.Join("uploads", clean)); err != nil && !os.IsNotExist(err) {
		log.Println("could not delete upload:", err)
	}
}
