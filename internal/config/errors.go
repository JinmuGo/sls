package config

import "errors"

var (
	ErrSSHDirNotExist    = errors.New("ssh directory does not exist")
	ErrSSHConfigNotExist = errors.New("ssh config file does not exist")
	ErrSSHConfigEmpty    = errors.New("no host entries in ssh config")
)
