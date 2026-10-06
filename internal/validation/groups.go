package validation

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"social/internal/models"
	"strings"
)

func ValidateGroup(g models.Group, image *multipart.FileHeader) error {
	if len(g.Description) > 200 {
		return errors.New("description length cannot be more then 200 character")
	}

	if len(g.Description) == 0 {
		return errors.New("group must have a description")
	}

	if len(g.Title) > 15 {
		return errors.New("name length cannot be more then 15 character")
	}

	if len(g.Title) == 0 {
		return errors.New("name must have a description")
	}

	if image != nil {
		allowedExtensions := map[string]bool{
			".png":  true,
			".jpg":  true,
			".jpeg": true,
			".gif":  true,
		}

		extension := strings.ToLower(filepath.Ext(image.Filename))

		if !allowedExtensions[extension] {
			return errors.New("image must be png, jpg or gif")
		}

		if image.Size <= 0 {
			return errors.New("invalid image")
		}
	}

	return nil
}
