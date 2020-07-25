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

func New() (*Target, error) {
	return &Target{}, nil
}
