package repository

import "errors"

var (
	ErrUserExists       = errors.New("user already exists")
	ErrUserNotFound     = errors.New("user not found")
	ErrLeadNotFound     = errors.New("lead not found")
	ErrDealNotFound     = errors.New("deal not found")
	ErrPropertyNotFound = errors.New("property not found")
	ErrNoFieldsToUpdate = errors.New("no fields to update")
)
