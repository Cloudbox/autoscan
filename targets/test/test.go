package test

import (
	"fmt"

	"github.com/cloudbox/autoscan"
)

type Target struct{}

func (t Target) Scan(scans []autoscan.Scan) error {
	fmt.Println(scans)
	return nil
}

func (t Target) Available() bool {
	return true
}

func New() (*Target, error) {
	return &Target{}, nil
}
