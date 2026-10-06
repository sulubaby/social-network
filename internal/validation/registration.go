package validation

import (
	"errors"
	"regexp"
	"social/internal/models"
	"strings"
	"time"
)

func ValidateRegisterData(userData *models.UserRegistration) error {
	NormalizeRegisterData(userData)

	if err := validateNames(&userData.FirstName); err != nil {
		return err
	}
	if err := validateNames(&userData.LastName); err != nil {
		return err
	}

	if err := validateUserName(&userData.UserName); err != nil {
		return err
	}

	if err := validateEmail(&userData.Email); err != nil {
		return err
	}

	if err := validateDOB(&userData.DOB); err != nil {
		return err
	}

	if err := validatePassword(userData.Password); err != nil {
		return err
	}

	if err := validateAbout(&userData.About); err != nil {
		return nil
	}

	return nil
}

// take the data and normilize it for the database and the validation as removing trail and leading spaces and capitalize or lowerize the chracters
func NormalizeRegisterData(userData *models.UserRegistration) {
	// trim spaces
	userData.FirstName = strings.Trim(userData.FirstName, " ")
	userData.LastName = strings.Trim(userData.LastName, " ")
	userData.UserName = strings.Trim(userData.UserName, " ")
	userData.Email = strings.Trim(userData.Email, " ")
	userData.Password = strings.Trim(userData.Password, " ")
	userData.About = strings.Trim(userData.About, " ")

	// normalize
	if userData.FirstName != "" && len(userData.FirstName) >= 2 {
		userData.FirstName = strings.ToUpper(userData.FirstName[:1]) + userData.FirstName[1:]
	}
	if userData.LastName != "" && len(userData.LastName) >= 2 {
		userData.LastName = strings.ToUpper(userData.LastName[:1]) + userData.LastName[1:]
	}
	if userData.UserName != "" {
		userData.UserName = strings.ToLower(userData.UserName)
	}
	if userData.Email != "" {
		userData.Email = strings.ToLower(userData.Email)
	}
	

}

func validateNames(name *string) error {
	if len(*name) < 2 || len(*name) > 15 {
		return errors.New("Error: first/last name must be between 3 and 15 characters")
	}

	nameReg, err := regexp.Compile(`^[A-Z][a-zA-Z]*$`)
	if err != nil {
		return err
	}

	if !nameReg.MatchString(*name) {
		return errors.New("Error: name format is incorrect")
	}

	return nil
}

func validateUserName(username *string) error {
	if len(*username) == 0 {
		return nil
	}
	if len(*username) < 3 || len(*username) > 12 {
		return errors.New("Error: usernam length must be between 3 and 12 characters")
	}

	usernameReq, err := regexp.Compile("^[a-z0-9_-]+$")
	if err != nil {
		return err
	}

	if !usernameReq.MatchString(*username) {
		return errors.New("Error: invalid username format")
	}

	return nil
}

func validateEmail(email *string) error {
	if len(*email) < 5 || len(*email) > 75 {
		return errors.New("Error: email length must be between 5 and 75 characters")
	}

	emailReg, err := regexp.Compile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if err != nil {
		return err
	}

	if !emailReg.MatchString(*email) {
		return errors.New("Error: invalid email")
	}

	return nil
}

func validateDOB(dob *time.Time) error {
	now := time.Now()

	if dob.After(now) {
		return errors.New("date of birth is invalid")
	}

	tooOld := now.AddDate(-120, 0, 0)

	if dob.Before(tooOld) {
		return errors.New("date of birth is invalid")
	}

	tooYoung := now.AddDate(-12, 0, 0)
	if dob.After(tooYoung) {
		return errors.New("date of birth is invalid")
	}

	return nil
}

func validatePassword(pass string) error {
	if len(pass) < 8 {
		return errors.New("invalid password: password must be at least 8 characters long")
	}

	if len(pass) > 75 {
		return errors.New("invalid password: password must be less than 75 characters long")
	}

	numberCheck, err := regexp.MatchString(`[0-9]`, pass)
	if err != nil {
		return errors.New("invalid password: password format is wrong")
	}

	if !numberCheck {
		return errors.New("invalid password: password must contain a number")
	}

	specialCharacterCheck, err := regexp.MatchString(`[!@#$%*&^?]`, pass)
	if err != nil {
		return errors.New("invalid password: password format is wrong")
	}

	if !specialCharacterCheck {
		return errors.New("invalid password: password must contain a special character: ! @ # $ % * & ^ ?")
	}

	alphaCheck, err := regexp.MatchString(`[a-zA-Z]`, pass)
	if err != nil {
		return errors.New("invalid password: password format is wrong")
	}

	if !alphaCheck {
		return errors.New("invalid password: password must contain alphabetic characters")
	}

	forbiddenCheck, err := regexp.MatchString(`[^a-zA-Z0-9!@#$%*&^?]`, pass)
	if err != nil {
		return errors.New("invalid password: password format is wrong")
	}

	if forbiddenCheck {
		return errors.New("invalid password: password contains forbidden characters")
	}

	return nil
}

func validateAbout(about *string) error {
	if len(*about) == 0 {
		return nil
	}
	if len(*about) > 1000 {
		return errors.New("Error: about is too long")
	}
	return nil
}
