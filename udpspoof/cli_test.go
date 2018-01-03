package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixWindowsIPConfigMAC(t *testing.T) {
	in := "AB-CD-EF-01-02-03"
	out := standardizeMACFormat(in)
	expectedOut := "AB:CD:EF:01:02:03"
	assert.Equal(t, expectedOut, out)
}

func TestFixMacosArpMAC(t *testing.T) {
	in := "1:2:FF:4:05:6"
	out := standardizeMACFormat(in)
	expectedOut := "01:02:FF:04:05:06"
	assert.Equal(t, expectedOut, out)
}
